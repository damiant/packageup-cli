# PackageUp

File upload and download service backed by Cloudflare Workers and R2.

## Project Structure

```
serve/      Cloudflare Worker API
upload/     Go CLI for uploading files
download/   Go CLI for downloading files
```

## serve

Cloudflare Worker that handles file storage in the `packageup-static` R2 bucket.

### Prerequisites

- Node.js
- [Wrangler CLI](https://developers.cloudflare.com/workers/wrangler/install-and-update/)
- A Cloudflare account with the `packageup-static` R2 bucket created

### Setup

```sh
cd serve
npm install
```

### Development

```sh
npm run dev
```

### Deploy

```sh
npm run deploy
```

### Regenerate Types

After changing bindings in `wrangler.jsonc`:

```sh
npm run cf-typegen
```

### API

| Method   | Path        | Params                                         | Description                |
|----------|-------------|-------------------------------------------------|----------------------------|
| `POST`   | `/upload`   | —                                               | Simple upload (< 100MB)    |
| `POST`   | `/upload`   | `?action=mpu-create`                            | Start multipart upload     |
| `PUT`    | `/upload`   | `?filename=&uploadId=&partNumber=`              | Upload a part              |
| `POST`   | `/upload`   | `?action=mpu-complete&filename=&uploadId=`      | Complete multipart upload  |
| `DELETE` | `/upload`   | `?filename=&uploadId=`                          | Abort multipart upload     |
| `GET`    | `/download` | `?filename=`                                    | Download a file            |

## upload

Go CLI that uploads a local file to the Worker API. Files under 10MB use a single request. Larger files are split into 10MB chunks and uploaded in parallel.

### Prerequisites

- Go 1.22+

### Build

```sh
cd upload
go build -o upload .
```

### Usage

```sh
./upload <path-to-file>
```

The CLI prints the generated 6-character filename on success:

```
uploaded: a3x9k2
```

### Configuration

Edit the `endpoint` constant in `main.go` to point to your deployed Worker URL.

## download

Go CLI that downloads a file from the Worker API using the 6-character filename returned by the upload CLI.

### Prerequisites

- Go 1.22+

### Build

```sh
cd download
go build -o download .
```

### Usage

```sh
# Save with the original filename
./download <filename>

# Save to a specific path
./download <filename> output.tar.gz
```

### Configuration

Edit the `endpoint` constant in `main.go` to point to your deployed Worker URL.
