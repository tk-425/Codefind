package indexer

const ManifestSchemaVersion = 1

type Manifest struct {
	SchemaVersion int                     `json:"schema_version"`
	RepoID        string                  `json:"repo_id"`
	OrgID         string                  `json:"org_id"`
	Files         map[string]ManifestFile `json:"files"`
}

type ManifestFile struct {
	Path               string `json:"path"`
	ContentHash        string `json:"content_hash,omitempty"`
	LastIndexedCommit  string `json:"last_indexed_commit,omitempty"`
	LastChunkingMethod string `json:"last_chunking_method,omitempty"`
	FallbackReason     string `json:"fallback_reason,omitempty"`
	ChunkingVersion    string `json:"chunking_version,omitempty"`
	LastIndexedAt      string `json:"last_indexed_at,omitempty"`
}
