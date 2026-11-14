package playwell

// ServerStatus represents the root handshake response from GET /.
type ServerStatus struct {
	Status                string            `json:"status"`
	Service               string            `json:"service"`
	Description           string            `json:"description"`
	Developer             string            `json:"developer"`
	SystemVersion         string            `json:"system_version"`
	AvailableAPIVersions  []string          `json:"available_api_versions"`
	RecommendedAPIVersion string            `json:"recommended_api_version"`
	SourceCode            string            `json:"source_code"`
	ContainerRegistry     string            `json:"container_registry"`
	Endpoints             map[string]string `json:"endpoints"`
}

// Kit represents metadata for a LEGO set.
type Kit struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Theme         string  `json:"theme"`
	Pieces        *int    `json:"pieces,omitempty"`
	Year          *int    `json:"year,omitempty"`
	AgeGroup      *string `json:"age_group,omitempty"`
	CoverURL      *string `json:"cover_url,omitempty"`
	IsCoverCached bool    `json:"is_cover_cached"`
}

// Instruction represents metadata for a LEGO instruction booklet.
type Instruction struct {
	ID            string   `json:"id"`
	KitID         string   `json:"kit_id"`
	Description   string   `json:"description"`
	PDFURL        string   `json:"pdf_url"`
	SizeMB        *float64 `json:"size_mb,omitempty"`
	CoverURL      *string  `json:"cover_url,omitempty"`
	IsPDFCached   bool     `json:"is_pdf_cached"`
	IsCoverCached bool     `json:"is_cover_cached"`
}

// KitSearchOptions defines optional query fitlers for SearchKits.
type KitSearchOptions struct {
	Search   string
	Name     string
	Theme    string
	AgeGroup string
}
