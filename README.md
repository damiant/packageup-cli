# PackageUp

File upload and download service backed by Cloudflare Workers and R2.

## Quick Start

### Pack

Archive and upload the current directory in one command:

```sh
curl -sSL https://api.packageup.io/pack | bash
```

This creates a `.tar.xz` archive of the current directory, uploads it to R2 storage, and outputs:

```
a3x9k2 was created

To unpack, run:
  curl -sSL https://api.packageup.io/unpack | bash -s a3x9k2
```

The 6-character ID (`a3x9k2`) is randomly generated and uniquely identifies the uploaded archive. No tools need to be pre-installed beyond `curl` and `tar`.

### Unpack

Download and extract a previously packed archive into the current directory:

```sh
curl -sSL https://api.packageup.io/unpack | bash -s <id>
```

Replace `<id>` with the 6-character filename returned by the pack command. The archive is downloaded, extracted with `tar -xJvf`, and the temporary file is cleaned up automatically.

```
a3x9k2 was unpacked
```

After a file is downloaded it is removed from storage.

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
| `GET`    | `/pack`     | —                                               | Serve pack script          |
| `GET`    | `/unpack`   | —                                               | Serve unpack script        |
| `GET`    | `/install`  | —                                               | Serve install script       |

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
