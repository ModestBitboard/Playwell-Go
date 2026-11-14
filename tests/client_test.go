package playwell_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	playwell "github.com/ModestBitboard/Playwell-Go"
)

func setupMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"detail": "Not found"})
			return
		}
		_ = json.NewEncoder(w).Encode(playwell.ServerStatus{
			Status:                "up",
			Service:               "Playwell API",
			Description:           "Mock Playwell Server",
			Developer:             "Bitboard",
			SystemVersion:         "1.0.2",
			AvailableAPIVersions:  []string{"v1"},
			RecommendedAPIVersion: "v1",
			SourceCode:            "https://github.com/ModestBitboard/playwell-server",
			ContainerRegistry:     "ghcr.io/modestbitboard/playwell-server",
			Endpoints:             map[string]string{"swagger_ui": "/docs"},
		})
	})

	mux.HandleFunc("GET /v1/kits/71374", func(w http.ResponseWriter, r *http.Request) {
		pieces := 2646
		year := 2020
		age := "18+"
		_ = json.NewEncoder(w).Encode(playwell.Kit{
			ID:            "71374",
			Name:          "Nintendo Entertainment System™",
			Theme:         "LEGO® Super Mario™",
			Pieces:        &pieces,
			Year:          &year,
			AgeGroup:      &age,
			IsCoverCached: true,
		})
	})

	mux.HandleFunc("GET /v1/kits/999999", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"detail": "Kit 999999 not found"})
	})

	mux.HandleFunc("GET /v1/instructions/6667243/pdf", func(w http.ResponseWriter, r *http.Request) {
		// simulate 307 redirect to CDN
		http.Redirect(w, r, "/cdn/manual.pdf", http.StatusTemporaryRedirect)
	})

	mux.HandleFunc("GET /cdn/manual.pdf", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4 fake binary manual payload"))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestHandshakeAndGetKit(t *testing.T) {
	srv := setupMockServer(t)
	ctx := context.Background()

	client, err := playwell.NewClient(playwell.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	status, err := client.ServerStatus(ctx)
	if err != nil {
		t.Fatalf("unexpected server status: %+v, err=%v", status, err)
	}

	kit, err := client.GetKit(ctx, "71374")
	if err != nil {
		t.Fatalf("GetKit failed: %v", err)
	}
	if kit.Name != "Nintendo Entertainment System™" || *kit.Pieces != 2646 {
		t.Fatalf("unexpected kit metadata: %+v", kit)
	}
}

func TestNotFoundSentinalError(t *testing.T) {
	srv := setupMockServer(t)
	client, err := playwell.NewClient(playwell.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, err = client.GetKit(context.Background(), "999999")
	if !errors.Is(err, playwell.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestVersionMismatchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(playwell.ServerStatus{
			Status:               "up",
			AvailableAPIVersions: []string{"v2"},
		})
	}))
	defer srv.Close()

	_, err := playwell.NewClient(playwell.WithBaseURL(srv.URL))
	if !errors.Is(err, playwell.ErrVersionMismatch) {
		t.Fatalf("expected ErrVersionMismatch, got %v", err)
	}
}

func TestBinaryAssestsAndRedirects(t *testing.T) {
	srv := setupMockServer(t)
	ctx := context.Background()
	client, err := playwell.NewClient(playwell.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// 1. In-memory bytes
	raw, err := client.GetInstructionPDF(ctx, "6667243")
	if err != nil {
		t.Fatalf("GetInstructionPDF failed: %s", err)
	}
	if string(raw) != "%PDF-1.4 fake binary manual payload" {
		t.Errorf("unexpected PDF bytes: %q", string(raw))
	}

	// 2. Direct-to-disk download
	dest := filepath.Join(t.TempDir(), "nested", "manual.pdf")
	savedPath, err := client.DownloadInstructionPDF(ctx, "6667243", dest)
	if err != nil {
		t.Fatalf("DownloadInstructionPDF failed: %v", err)
	}
	diskBytes, err := os.ReadFile(savedPath)
	if err != nil || string(diskBytes) != string(raw) {
		t.Errorf("downloaded file mismatch: err=%v", err)
	}

	// 3. Raw io.ReadCloser stream
	rc, err := client.StreamInstructionPDF(ctx, "6667243")
	if err != nil {
		t.Fatalf("StreamInstructionPDF failed: %v", err)
	}
	defer rc.Close()
	streamed, _ := io.ReadAll(rc)
	if string(streamed) != string(raw) {
		t.Errorf("streamed bytes mismatch")
	}
}
