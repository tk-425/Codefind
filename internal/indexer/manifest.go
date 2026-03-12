package indexer

const ManifestSchemaVersion = 1

const (
	IndexModeHybrid      = "hybrid"
	IndexModeForceWindow = "force_window"
)

type Manifest struct {
	SchemaVersion int                     `json:"schema_version"`
	RepoID        string                  `json:"repo_id"`
	OrgID         string                  `json:"org_id"`
	RepoPath      string                  `json:"repo_path,omitempty"`
	InitializedAt string                  `json:"initialized_at,omitempty"`
	LastCommit    string                  `json:"last_commit,omitempty"`
	Files         map[string]ManifestFile `json:"files"`
}

type ManifestFile struct {
	Path               string   `json:"path"`
	ContentHash        string   `json:"content_hash,omitempty"`
	Language           string   `json:"language,omitempty"`
	SizeBytes          int64    `json:"size_bytes,omitempty"`
	LineCount          int      `json:"line_count,omitempty"`
	LastIndexedCommit  string   `json:"last_indexed_commit,omitempty"`
	LastModTime        string   `json:"last_mod_time,omitempty"`
	LastIndexMode      string   `json:"last_index_mode,omitempty"`
	LastChunkingMethod string   `json:"last_chunking_method,omitempty"`
	FallbackReason     string   `json:"fallback_reason,omitempty"`
	ChunkingVersion    string   `json:"chunking_version,omitempty"`
	LastIndexedAt      string   `json:"last_indexed_at,omitempty"`
	ChunkIDs           []string `json:"chunk_ids,omitempty"`
}
