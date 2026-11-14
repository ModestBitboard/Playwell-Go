# Playwell-Go

![Playwell Version](https://img.shields.io/badge/playwell-v1-red)
![GitHub License](https://img.shields.io/github/license/ModestBitboard/playwell-go?logo=gplv3&logoColor=white)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/ModestBitboard/playwell-go?logo=go&logoColor=white)


The official synchronous Go client SDK for [Playwell](https://github.com/ModestBitboard/playwell-server), a read-through caching REST API for LEGO kit metadata, cover artwork, and instruction manuals.

## Features

* **Zero External Dependencies:** Built exclusively on the Go standard library (`net/http`, `context`, `encoding/json`).
* **Config-less Default:** Connects automatically to the public instance at `https://playwell.bitboard.ca`, with functional options and environment overrides for self-hosted servers.
* **Memory, Disk, & Raw Streams:** Load cover images and PDFs directly into RAM (`[]byte`), stream heavy manuals straight to disk, or consume raw `io.ReadCloser` streams.
* **Automatic Handshake Validation:** Verifies server reachability and API version compatibility (`v1`) on initialization (can be disabled).
* **Idiomatic Error Handling:** Returns strongly typed structs (`Kit`, `Instruction`, `ServerStatus`) alongside sentinel errors (`ErrNotFound`, `ErrVersionMismatch`, `ErrConnection`) and structured `*APIError` types.


## Installation

```bash
go get github.com/ModestBitboard/Playwell-Go
```

## Quickstart

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	playwell "github.com/ModestBitboard/Playwell-Go"
)

func main() {
	ctx := context.Background()

	// Connects to https://playwell.bitboard.ca by default
	client, err := playwell.NewClient()
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}

	// Check server handshake & version info
	status, _ := client.ServerStatus(ctx)
	fmt.Printf("Connected to %s v%s\n", status.Service, status.SystemVersion)

	// Fetch kit metadata (triggers a server-side scrape if not yet cached)
	kit, err := client.GetKit(ctx, "71374")
	if errors.Is(err, playwell.ErrNotFound) {
		log.Fatal("LEGO kit not found")
	} else if err != nil {
		log.Fatalf("API error: %v", err)
	}

	fmt.Printf("Kit: %s (%s) - %d pieces\n", kit.Name, kit.Theme, *kit.Pieces)

	// List available instruction manuals for the kit
	manuals, err := client.GetKitInstructions(ctx, "71374")
	if err != nil {
		log.Fatalf("Failed to list instructions: %v", err)
	}

	for _, m := range manuals {
		fmt.Printf(" - [%s] %s (%.2f MB)\n", m.ID, m.Description, *m.SizeMB)
	}

	// Stream the first PDF directly to disk
	if len(manuals) > 0 {
		saved, err := client.DownloadInstructionPDF(ctx, manuals[0].ID, fmt.Sprintf("./%s_manual.pdf", kit.ID))
		if err != nil {
			log.Fatalf("Download failed: %v", err)
		}
		fmt.Printf("Saved manual to %s\n", saved)
	}
}
```

## Configuration

You can point the client at a self-hosted server either by passing `WithBaseURL(...)` to `playwell.NewClient(...)` or by setting the `PLAYWELL_SERVER_URL` environment variable:

```go
func configure() {
	// Option 1: Explicit functional options
	client, _ := playwell.NewClient(
		playwell.WithBaseURL("http://localhost:5346"),
		playwell.WithTimeout(30*time.Second),
	)

	// Option 2: Environment variable
	os.Setenv("PLAYWELL_SERVER_URL", "http://localhost:5346")
	client, _ = playwell.NewClient()

	// Option 3: Skip startup version handshake
	client, _ = playwell.NewClient(
		playwell.WithoutVersionCheck(),
	)
}
```

## In-Memory Assets & Raw Stream Assets

For GUI applications or web services that want to render artwork without creating temporary files, use the `Get*` binary methods to fetch `[]byte` directly into RAM, or `Stream*` to consume an `io.ReadCloser`:

```go
func loadAssets(ctx context.Context, client *playwell.Client) error {
	// Fetch directly into RAM
	coverBytes, err := client.GetKitCover(ctx, "71374")
	if err != nil {
		return err
	}
	reader := bytes.NewReader(coverBytes)
	_ = reader

	// Or open a raw streaming reader (caller must close)
	stream, err := client.StreamInstructionPDF(ctx, "6345102")
	if err != nil {
		return err
	}
	defer stream.Close()

	_, err = io.Copy(io.Discard, stream)
	return err
}
```

## API Reference

| Method                                       | Description                                                        | Return Type                |
|:---------------------------------------------|:-------------------------------------------------------------------|:---------------------------|
| `GetStatus(ctx)`                             | Fetches the latest server status handshake.                        | `(*ServerStatus, error)`   |
| `ServerStatus(ctx)`                          | Returns the cached server status handshake (fetches if unset).     | `(*ServerStatus, error)`   |
| `GetKit(ctx, kitID)`                         | Fetches full metadata for a LEGO kit ID.                           | `(*Kit, error)`            |
| `SearchKits(ctx, opts)`                      | Searches cached kits using `*KitSearchOptions`.                    | `([]Kit, error)`           |
| `GetKitInstructions(ctx, kitID)`             | Lists all instruction booklets for a kit ID.                       | `([]Instruction, error)`   |
| `GetInstruction(ctx, instructionID)`         | Fetches metadata for a standalone instruction ID.                  | `(*Instruction, error)`    |
| `SearchInstructions(ctx, description)`       | Searches cached instruction booklets by description.               | `([]Instruction, error)`   |
| `GetKitCover(ctx, kitID)`                    | Fetches kit box artwork into RAM.                                  | `([]byte, error)`          |
| `DownloadKitCover(ctx, kitID, path)`         | Streams kit box artwork directly to disk.                          | `(string, error)`          |
| `StreamKitCover(ctx, kitID)`                 | Opens a raw `io.ReadCloser` stream for kit box artwork.            | `(io.ReadCloser, error)`   |
| `GetInstructionCover(ctx, id)`               | Fetches booklet cover artwork into RAM.                            | `([]byte, error)`          |
| `DownloadInstructionCover(ctx, id, path)`    | Streams booklet cover artwork directly to disk.                    | `(string, error)`          |
| `StreamInstructionCover(ctx, id)`            | Opens a raw `io.ReadCloser` stream for booklet cover artwork.      | `(io.ReadCloser, error)`   |
| `GetInstructionPDF(ctx, id)`                 | Fetches instruction manual PDF into RAM.                           | `([]byte, error)`          |
| `DownloadInstructionPDF(ctx, id, path)`      | Streams instruction manual PDF directly to disk.                   | `(string, error)`          |
| `StreamInstructionPDF(ctx, id)`              | Opens a raw `io.ReadCloser` stream for an instruction manual PDF.  | `(io.ReadCloser, error)`   |

## Errors

All sentinel errors can be checked using `errors.Is`, and HTTP error details can be inspected using `errors.As` with `*playwell.APIError`:

* `ErrConnection` — Network timeout or unreachable server.
* `ErrVersionMismatch` — Server does not support the client's API version (`v1`).
* `ErrNotFound` — Requested kit or instruction ID returned HTTP 404.
* `*APIError` — Unexpected 4xx / 5xx HTTP error response (`StatusCode` and `Detail`).

## Related Projects

* [playwell-server](https://github.com/ModestBitboard/playwell-server) - The FastAPI backend server.
* [playwell-py](https://github.com/ModestBitboard/playwell-py) - The official Python SDK.
* [playwell-go](https://github.com/ModestBitboard/playwell-go) - The official Go SDK.

## License
Distributed under the GPLv3 License. See [LICENSE](https://github.com/ModestBitboard/Playwell-Go/blob/main/LICENSE) for more information.

## Support
If you liked this project, go ahead and give it a star! And if it really helped you out, consider sending me a tip.

> **BTC**: `bc1q8tmwuyxpgkuptwu3mn4ryaemp3umvwcc0e5nkk`

