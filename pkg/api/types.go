package api

type HealthResponse struct {
	Status       string `json:"status"`
	OllamaStatus string `json:"ollama_status"`
	QdrantStatus string `json:"qdrant_status"`
	Timestamp    string `json:"timestamp,omitempty"`
}
