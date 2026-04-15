package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const endpoint = "https://api.packageup.io/download"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: download <filename>\n")
		os.Exit(1)
	}
	name := os.Args[1]

	outPath := name
	if len(os.Args) >= 3 {
		outPath = os.Args[2]
	}

	if err := download(name, outPath); err != nil {
		fmt.Fprintf(os.Stderr, "download failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("saved: %s\n", outPath)
}

func download(name, outPath string) error {
	resp, err := http.Get(endpoint + "?filename=" + name)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("file %q not found", name)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("downloaded %d bytes\n", written)
	return nil
}
