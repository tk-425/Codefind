// Package api defines the shared data types for client-server communication.
//
// The Codefind API uses these types to serialize requests and responses over HTTP.
// All endpoints return JSON responses with an optional "error" field for error handling.
//
// Endpoints:
// - POST /tokenize - Count tokens in text using Ollama
// - POST /embed - Generate embeddings for text
// - POST /index - Index chunks and store in ChromaDB
// - POST /query - Search indexed chunks
// - GET /health - Check server health (Ollama + ChromaDB)
package api

import "time"

// ChunkMetadata contains all metadata for a code chunk stored in ChromaDB
type ChunkMetadata struct {
	// Repository identification
	RepoID      string `json:"repo_id"`
	ProjectName string `json:"project_name"`
	FilePath    string `json:"file_path"`
	Language    string `json:"language"`

	// Line information
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`

	// Content tracking
	ContentHash string `json:"content_hash"`
	ChunkTokens int    `json:"chunk_tokens"`

	// Embedding model
	ModelID string `json:"model_id"`

	// Indexing timestamp
	IndexedAt time.Time `json:"indexed_at"`

	// Symbol information (from LSP)
	SymbolName string `json:"symbol_name,omitempty"`
	SymbolKind string `json:"symbol_kind,omitempty"`

	// Split tracking
	IsSplit    bool `json:"is_split"`
	SplitIndex int  `json:"split_index,omitempty"`

	// Lifecycle status
	Status          string     `json:"status"` // "active" or "deleted"
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	DeletedInCommit string     `json:"deleted_in_commit,omitempty"`
}

// Chunk represents a code chunk with content and metadata
type Chunk struct {
	ID       string        `json:"id,omitempty"` // Stable chunk ID (SHA256-based)
	Content  string        `json:"content"`
	Metadata ChunkMetadata `json:"metadata"`
}

// --- Tokenize Endpoint ---

type TokenizeRequest struct {
	Model string   `json:"model"` // e.g., "unclemusclez/jina-embeddings-v2-base-code"
	Input []string `json:"input"` // Text to tokenize
}

type TokenizeResponse struct {
	Tokens []int  `json:"tokens"` // Token counts for each input
	Error  string `json:"error,omitempty"`
}

// --- Embed Endpoint ---

type EmbedRequest struct {
	Model string   `json:"model"` // e.g., "mxbai-embed-large"
	Input []string `json:"input"` // Text to embed
}

type EmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"` // Vector embeddings for each input
	Error      string      `json:"error,omitempty"`
}

// --- Index Endpoint ---

type IndexRequest struct {
	AuthKey    string  `json:"auth_key"` // For authentication
	Chunks     []Chunk `json:"chunks"`
	Collection string  `json:"collection"` // ChromaDB collection name
}

type IndexResponse struct {
	InsertedCount int    `json:"inserted_count"`
	UpdatedCount  int    `json:"updated_count"`
	Error         string `json:"error,omitempty"`
}

// --- Query Endpoint ---

type QueryRequest struct {
	Query      string            `json:"query"`               // Search query text
	Collection string            `json:"collection"`          // ChromaDB collection to search
	TopK       int               `json:"top_k"`               // Number of results
	Filters    map[string]string `json:"filters,omitempty"`   // Metadata filters
	Page       int               `json:"page,omitempty"`      // Pagination: page number (default 1)
	PageSize   int               `json:"page_size,omitempty"` // Pagination: results per page
}

type QueryResult struct {
	ChunkID    string        `json:"chunk_id"`
	Content    string        `json:"content"`
	Metadata   ChunkMetadata `json:"metadata"`
	Distance   float32       `json:"distance"`   // Similarity distance (lower is better)
	Similarity float32       `json:"similarity"` // Similarity score 0-1 (higher is better)
}

type QueryResponse struct {
	Results    []QueryResult `json:"results"`
	TotalCount int           `json:"total_count"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	Error      string        `json:"error,omitempty"`
}

// --- Health Endpoint ---

type HealthResponse struct {
	Status         string    `json:"status"`          // "ok" or "error"
	OllamaStatus   string    `json:"ollama_status"`   // "ok" or "error"
	ChromaDBStatus string    `json:"chromadb_status"` // "ok" or "error"
	Timestamp      time.Time `json:"timestamp"`
	Error          string    `json:"error,omitempty"`
}
