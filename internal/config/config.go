package config

type Config struct {
	ServerURL   string `json:"server_url"`
	ActiveOrgID string `json:"active_org_id"`
	Editor      string `json:"editor"`
}
