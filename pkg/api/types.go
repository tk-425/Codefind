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
