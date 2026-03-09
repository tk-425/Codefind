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
	RepoID     string      `json:"repo_id,omitempty"`
	RepoCount  int         `json:"repo_count"`
	ChunkCount int         `json:"chunk_count"`
	Repos      []RepoStats `json:"repos"`
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
