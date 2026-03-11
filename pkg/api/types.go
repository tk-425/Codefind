package api

type HealthResponse struct {
	Status       string `json:"status"`
	OllamaStatus string `json:"ollama_status"`
	QdrantStatus string `json:"qdrant_status"`
	Timestamp    string `json:"timestamp,omitempty"`
}

type OrgSummary struct {
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name,omitempty"`
	OrganizationSlug string `json:"organization_slug,omitempty"`
	Role             string `json:"role"`
}

type OrgListResponse struct {
	Data       []OrgSummary `json:"data"`
	TotalCount int          `json:"total_count"`
}

type OrganizationMember struct {
	UserID          string `json:"user_id"`
	MembershipID    string `json:"membership_id,omitempty"`
	Role            string `json:"role"`
	FirstName       string `json:"first_name,omitempty"`
	LastName        string `json:"last_name,omitempty"`
	EmailAddress    string `json:"email_address,omitempty"`
	ProfileImageURL string `json:"profile_image_url,omitempty"`
}

type OrganizationMemberListResponse struct {
	Data       []OrganizationMember `json:"data"`
	TotalCount int                  `json:"total_count"`
}

type OrganizationInvitation struct {
	ID             string `json:"id"`
	InvitationID   string `json:"invitation_id"`
	EmailAddress   string `json:"email_address"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	OrganizationID string `json:"organization_id"`
	CreatedAt      int64  `json:"created_at,omitempty"`
	UpdatedAt      int64  `json:"updated_at,omitempty"`
	ExpiresAt      int64  `json:"expires_at,omitempty"`
	InviterUserID  string `json:"inviter_user_id,omitempty"`
}

type OrganizationInvitationListResponse struct {
	Data       []OrganizationInvitation `json:"data"`
	TotalCount int                      `json:"total_count"`
}

type CreateOrganizationInvitationRequest struct {
	EmailAddress string `json:"email_address"`
	Role         string `json:"role"`
}

type CollectionSummary struct {
	RepoID string `json:"repo_id"`
}

type CollectionListResponse struct {
	Data       []CollectionSummary `json:"data"`
	TotalCount int                 `json:"total_count"`
}

type RepoStats struct {
	RepoID     string `json:"repo_id"`
	ChunkCount int    `json:"chunk_count"`
}

type StatsResponse struct {
	RepoID          string      `json:"repo_id,omitempty"`
	RepoCount       int         `json:"repo_count"`
	ChunkCount      int         `json:"chunk_count"`
	ActiveChunks    int         `json:"active_chunks"`
	DeletedChunks   int         `json:"deleted_chunks"`
	TotalChunks     int         `json:"total_chunks"`
	OverheadPercent float64     `json:"overhead_percent"`
	Repos           []RepoStats `json:"repos"`
}

type QueryRequest struct {
	QueryText string `json:"query_text"`
	RepoID    string `json:"repo_id,omitempty"`
	Project   string `json:"project,omitempty"`
	Language  string `json:"language,omitempty"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	TopK      int    `json:"top_k"`
}

type QueryResult struct {
	ID        string  `json:"id"`
	Score     float64 `json:"score"`
	RepoID    string  `json:"repo_id"`
	Project   string  `json:"project,omitempty"`
	Language  string  `json:"language,omitempty"`
	Path      string  `json:"path,omitempty"`
	Snippet   string  `json:"snippet,omitempty"`
	Content   string  `json:"content,omitempty"`
	Page      int     `json:"page,omitempty"`
	StartLine int     `json:"start_line,omitempty"`
	EndLine   int     `json:"end_line,omitempty"`
}

type QueryResponse struct {
	Data       []QueryResult `json:"data"`
	TotalCount int           `json:"total_count"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	HasMore    bool          `json:"has_more"`
}

type TokenizeRequest struct {
	Text string `json:"text"`
}

type TokenizeResponse struct {
	Model      string   `json:"model"`
	Tokens     []string `json:"tokens"`
	TokenCount int      `json:"token_count"`
}

type ChunkMetadata struct {
	RepoID         string `json:"repo_id"`
	Path           string `json:"path"`
	Language       string `json:"language,omitempty"`
	StartLine      int    `json:"start_line"`
	EndLine        int    `json:"end_line"`
	ContentHash    string `json:"content_hash"`
	Status         string `json:"status"`
	SymbolName     string `json:"symbol_name,omitempty"`
	SymbolKind     string `json:"symbol_kind,omitempty"`
	ParentName     string `json:"parent_name,omitempty"`
	IndexedAt      string `json:"indexed_at,omitempty"`
	ChunkingMethod string `json:"chunking_method,omitempty"`
	FallbackReason string `json:"fallback_reason,omitempty"`
}

type IndexChunk struct {
	ID       string        `json:"id"`
	Content  string        `json:"content"`
	Metadata ChunkMetadata `json:"metadata"`
}

type IndexRequest struct {
	RepoID string       `json:"repo_id"`
	Chunks []IndexChunk `json:"chunks"`
}

type IndexResponse struct {
	Status       string `json:"status"`
	RepoID       string `json:"repo_id"`
	IndexedCount int    `json:"indexed_count"`
	Accepted     bool   `json:"accepted"`
	Detail       string `json:"detail,omitempty"`
}

type ChunkStatusUpdateRequest struct {
	RepoID   string   `json:"repo_id"`
	ChunkIDs []string `json:"chunk_ids"`
	Status   string   `json:"status"`
}

type ChunkStatusUpdateResponse struct {
	Status       string `json:"status"`
	RepoID       string `json:"repo_id"`
	UpdatedCount int    `json:"updated_count"`
	Detail       string `json:"detail,omitempty"`
}

type TombstonedChunkSummary struct {
	Path         string `json:"path"`
	ChunkCount   int    `json:"chunk_count"`
	TombstonedAt string `json:"tombstoned_at,omitempty"`
}

type TombstonedChunkListResponse struct {
	Status     string                   `json:"status"`
	RepoID     string                   `json:"repo_id"`
	FoundCount int                      `json:"found_count"`
	Files      []TombstonedChunkSummary `json:"files"`
	Detail     string                   `json:"detail,omitempty"`
}

type ChunkPurgeRequest struct {
	RepoID        string `json:"repo_id"`
	OlderThanDays int    `json:"older_than_days"`
}

type ChunkPurgeResponse struct {
	Status      string                   `json:"status"`
	RepoID      string                   `json:"repo_id"`
	FoundCount  int                      `json:"found_count"`
	PurgedCount int                      `json:"purged_count"`
	Files       []TombstonedChunkSummary `json:"files"`
	Detail      string                   `json:"detail,omitempty"`
}

type RepoClearRequest struct {
	RepoID string `json:"repo_id"`
}

type RepoClearResponse struct {
	Status  string `json:"status"`
	RepoID  string `json:"repo_id"`
	Cleared bool   `json:"cleared"`
	Detail  string `json:"detail,omitempty"`
}
