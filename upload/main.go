package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
)

const (
	partSize    = 10 * 1024 * 1024 // 10MB per part
	maxParallel = 4
	endpoint    = "https://serve.packageup.workers.dev/upload"
)

type createResponse struct {
	Filename string `json:"filename"`
	UploadID string `json:"uploadId"`
}

type uploadedPart struct {
	PartNumber int    `json:"partNumber"`
	Etag       string `json:"etag"`
}

type completeBody struct {
	Parts []uploadedPart `json:"parts"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: upload <filename>\n")
		os.Exit(1)
	}
	filepath := os.Args[1]

	info, err := os.Stat(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fileSize := info.Size()

	// For small files, use simple upload
	if fileSize <= partSize {
		filename, err := simpleUpload(filepath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "upload failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("uploaded: %s\n", filename)
		return
	}

	// Multipart upload for large files
	filename, err := multipartUpload(filepath, fileSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "upload failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("uploaded: %s\n", filename)
}

func simpleUpload(filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	req, err := http.NewRequest("POST", endpoint, f)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", readError(resp)
	}

	var result createResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	return result.Filename, nil
}

func multipartUpload(filepath string, fileSize int64) (string, error) {
	// Step 1: Create multipart upload
	resp, err := http.Post(endpoint+"?action=mpu-create", "", nil)
	if err != nil {
		return "", fmt.Errorf("create failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create failed: %w", readError(resp))
	}

	var created createResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("failed to decode create response: %w", err)
	}

	filename := created.Filename
	uploadID := created.UploadID
	totalParts := int((fileSize + partSize - 1) / partSize)

	fmt.Printf("uploading %s (%d bytes, %d parts)\n", filepath, fileSize, totalParts)

	// Step 2: Upload parts in parallel
	parts := make([]uploadedPart, totalParts)
	errs := make([]error, totalParts)
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup

	for i := 0; i < totalParts; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(partIndex int) {
			defer wg.Done()
			defer func() { <-sem }()

			offset := int64(partIndex) * partSize
			size := partSize
			if remaining := fileSize - offset; remaining < int64(size) {
				size = int(remaining)
			}

			part, err := uploadPart(filepath, filename, uploadID, partIndex+1, offset, size)
			if err != nil {
				errs[partIndex] = err
				return
			}
			parts[partIndex] = part
			fmt.Printf("  part %d/%d done\n", partIndex+1, totalParts)
		}(i)
	}
	wg.Wait()

	// Check for errors
	for i, e := range errs {
		if e != nil {
			// Abort on first error
			abortUpload(filename, uploadID)
			return "", fmt.Errorf("part %d failed: %w", i+1, e)
		}
	}

	// Step 3: Complete upload
	body := completeBody{Parts: parts}
	bodyBytes, _ := json.Marshal(body)

	url := fmt.Sprintf("%s?action=mpu-complete&filename=%s&uploadId=%s", endpoint, filename, uploadID)
	completeResp, err := http.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("complete failed: %w", err)
	}
	defer completeResp.Body.Close()

	if completeResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("complete failed: %w", readError(completeResp))
	}

	return filename, nil
}

func uploadPart(filepath, filename, uploadID string, partNumber int, offset int64, size int) (uploadedPart, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return uploadedPart{}, err
	}
	defer f.Close()

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return uploadedPart{}, err
	}

	reader := io.LimitReader(f, int64(size))

	url := fmt.Sprintf("%s?filename=%s&uploadId=%s&partNumber=%s",
		endpoint, filename, uploadID, strconv.Itoa(partNumber))

	req, err := http.NewRequest("PUT", url, reader)
	if err != nil {
		return uploadedPart{}, err
	}
	req.ContentLength = int64(size)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return uploadedPart{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return uploadedPart{}, readError(resp)
	}

	var part uploadedPart
	if err := json.NewDecoder(resp.Body).Decode(&part); err != nil {
		return uploadedPart{}, fmt.Errorf("failed to decode part response: %w", err)
	}
	return part, nil
}

func abortUpload(filename, uploadID string) {
	url := fmt.Sprintf("%s?filename=%s&uploadId=%s", endpoint, filename, uploadID)
	req, _ := http.NewRequest("DELETE", url, nil)
	http.DefaultClient.Do(req)
}

func readError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp errorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return fmt.Errorf("%s (HTTP %d)", errResp.Error, resp.StatusCode)
	}
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
}
