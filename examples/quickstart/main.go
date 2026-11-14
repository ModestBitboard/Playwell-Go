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
