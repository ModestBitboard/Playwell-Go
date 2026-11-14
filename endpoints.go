package playwell

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type fastapiError struct {
	Detail string `json:"detail"`
}

func parseAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var apiErr fastapiError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Detail != "" {
		return &APIError{
			StatusCode: resp.StatusCode,
			Detail:     apiErr.Detail,
		}
	}

	detail := strings.TrimSpace(string(body))
	if detail == "" {
		detail = http.StatusText(resp.StatusCode)
	}
	return &APIError{
		StatusCode: resp.StatusCode,
		Detail:     detail,
	}
}

// --- HELPERS ---

func (c *Client) getJSON(ctx context.Context, endpoint string, query url.Values, out any) error {
	reqURL := fmt.Sprintf("%s/%s/%s", c.baseURL, c.apiVersion, strings.TrimLeft(endpoint, "/"))
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnection, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrConnection, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("playwell: failed to decode response from %s: %w", endpoint, err)
	}

	return nil
}

func (c *Client) streamAsset(ctx context.Context, endpoint string) (io.ReadCloser, error) {
	reqURL := fmt.Sprintf("%s/%s/%s", c.baseURL, c.apiVersion, strings.TrimLeft(endpoint, "/"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnection, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnection, err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, parseAPIError(resp)
	}

	return resp.Body, nil
}

func (c *Client) getBytes(ctx context.Context, endpoint string) ([]byte, error) {
	rc, err := c.streamAsset(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("playwell: error reading binary stream from %s: %w", endpoint, err)
	}
	return data, nil
}

func (c *Client) downloadFile(ctx context.Context, endpoint, destPath string) (string, error) {
	rc, err := c.streamAsset(ctx, endpoint)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	absPath, err := filepath.Abs(destPath)
	if err != nil {
		return "", fmt.Errorf("playwell: invalid destination path %q: %w", destPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("playwell: failed to create directories for %q: %w", absPath, err)
	}

	f, err := os.Create(absPath)
	if err != nil {
		return "", fmt.Errorf("playwell: failed to create file %q: %w", absPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, rc); err != nil {
		return "", fmt.Errorf("playwell: failed writing stream to %q: %w", absPath, err)
	}

	return absPath, nil
}

// --- DATA ENDPOINTS ---

// GetKit fetches metadata for a specific LEGO kit ID (scrapes server-side if uncached).
func (c *Client) GetKit(ctx context.Context, kitID string) (*Kit, error) {
	var kit Kit
	if err := c.getJSON(ctx, "kits/"+url.PathEscape(kitID), nil, &kit); err != nil {
		return nil, err
	}
	return &kit, nil
}

// SearchKits queries cached kits matching the provided optional filter criteria.
func (c *Client) SearchKits(ctx context.Context, opts *KitSearchOptions) ([]Kit, error) {
	query := url.Values{}
	if opts != nil {
		if opts.Search != "" {
			query.Set("search", opts.Search)
		}
		if opts.Name != "" {
			query.Set("name", opts.Name)
		}
		if opts.Theme != "" {
			query.Set("theme", opts.Theme)
		}
		if opts.AgeGroup != "" {
			query.Set("age_group", opts.AgeGroup)
		}
	}

	var kits []Kit
	if err := c.getJSON(ctx, "kits", query, &kits); err != nil {
		return nil, err
	}
	return kits, nil
}

// GetKitInstructions lists all instruction booklets associated with a specific kit ID.
func (c *Client) GetKitInstructions(ctx context.Context, kitID string) ([]Instruction, error) {
	var instructions []Instruction
	if err := c.getJSON(ctx, fmt.Sprintf("kits/%s/instructions", url.PathEscape(kitID)), nil, &instructions); err != nil {
		return nil, err
	}
	return instructions, nil
}

// GetInstruction fetches metadata for a standalone instruction manual ID.
func (c *Client) GetInstruction(ctx context.Context, instructionID string) (*Instruction, error) {
	var instruction Instruction
	if err := c.getJSON(ctx, "instructions/"+url.PathEscape(instructionID), nil, &instruction); err != nil {
		return nil, err
	}
	return &instruction, nil
}

// SearchInstructions searches cached instruction booklets by description.
func (c *Client) SearchInstructions(ctx context.Context, description string) ([]Instruction, error) {
	query := url.Values{}
	if description != "" {
		query.Set("description", description)
	}

	var instructions []Instruction
	if err := c.getJSON(ctx, "instructions", query, &instructions); err != nil {
		return nil, err
	}
	return instructions, nil
}

// --- BINARY ENDPOINTS -> MEM ---

// GetKitCover fetches the kit's cover image and returns it as a byte slice.
func (c *Client) GetKitCover(ctx context.Context, kitID string) ([]byte, error) {
	return c.getBytes(ctx, fmt.Sprintf("kits/%s/cover", url.PathEscape(kitID)))
}

// GetInstructionCover fetches the instruction's cover image and returns it as a byte slice.
func (c *Client) GetInstructionCover(ctx context.Context, instructionID string) ([]byte, error) {
	return c.getBytes(ctx, fmt.Sprintf("instructions/%s/cover", url.PathEscape(instructionID)))
}

// GetInstructionPDF fetches the instruction's PDF and returns it as a byte slice.
func (c *Client) GetInstructionPDF(ctx context.Context, instructionID string) ([]byte, error) {
	return c.getBytes(ctx, fmt.Sprintf("instructions/%s/pdf", url.PathEscape(instructionID)))
}

// --- BINARY ENDPOINTS -> DISK ---

// DownloadKitCover streams the kit's cover image to a file.
func (c *Client) DownloadKitCover(ctx context.Context, kitID string, destPath string) (string, error) {
	return c.downloadFile(ctx, fmt.Sprintf("kits/%s/cover", url.PathEscape(kitID)), destPath)
}

// DownloadInstructionCover streams the instruction's cover image to a file.
func (c *Client) DownloadInstructionCover(ctx context.Context, instructionID string, destPath string) (string, error) {
	return c.downloadFile(ctx, fmt.Sprintf("instructions/%s/cover", url.PathEscape(instructionID)), destPath)
}

// DownloadInstructionCover streams the instruction's PDF to a file.
func (c *Client) DownloadInstructionPDF(ctx context.Context, instructionID string, destPath string) (string, error) {
	return c.downloadFile(ctx, fmt.Sprintf("instructions/%s/pdf", url.PathEscape(instructionID)), destPath)
}

// --- BINARY ENDPOINTS -> Stream ---

// StreamKitCover yields raw byte chunks of the kit's cover image for custom streaming pipelines.
func (c *Client) StreamKitCover(ctx context.Context, kitID string) (io.ReadCloser, error) {
	return c.streamAsset(ctx, fmt.Sprintf("kits/%s/cover", url.PathEscape(kitID)))
}

// StreamInstructionCover yields raw byte chunks of the instruction's cover image for custom streaming pipelines.
func (c *Client) StreamInstructionCover(ctx context.Context, instructionID string) (io.ReadCloser, error) {
	return c.streamAsset(ctx, fmt.Sprintf("instructions/%s/cover", url.PathEscape(instructionID)))
}

// StreamInstructionPDF yields raw byte chunks of an instruction PDF for custom streaming pipelines.
func (c *Client) StreamInstructionPDF(ctx context.Context, instructionID string) (io.ReadCloser, error) {
	return c.streamAsset(ctx, fmt.Sprintf("instructions/%s/pdf", url.PathEscape(instructionID)))
}
