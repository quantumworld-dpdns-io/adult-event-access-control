// TODO: Register this handler in cmd/server/main.go
package search

import "fmt"

// EventResult represents a search result from Qdrant.
type EventResult struct {
	ID    string  `json:"id"`
	Title string  `json:"title"`
	Score float64 `json:"score"`
}

// QdrantClient is a stub for the Qdrant vector search service.
// TODO: Replace placeholders with actual qdrant HTTP client integration
// using github.com/qdrant/go-client/qdrant.
type QdrantClient struct {
	Host   string
	Port   string
	APIKey string
}

// NewQdrantClient creates a new QdrantClient stub.
func NewQdrantClient(host, port, apiKey string) *QdrantClient {
	return &QdrantClient{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
	}
}

// SearchEvents returns placeholder results.
// TODO: Implement actual gRPC/HTTP call to Qdrant.
func (c *QdrantClient) SearchEvents(query string, limit int) ([]EventResult, error) {
	if query == "" {
		return nil, fmt.Errorf("search: query is empty")
	}
	// Placeholder: return empty results until Qdrant is wired in.
	return []EventResult{}, nil
}

// IndexEvent is a placeholder for inserting an event embedding into Qdrant.
// TODO: Implement actual gRPC/HTTP call to Qdrant upsert API.
func (c *QdrantClient) IndexEvent(eventID string, embedding []float32) error {
	if eventID == "" {
		return fmt.Errorf("search: eventID is empty")
	}
	if len(embedding) == 0 {
		return fmt.Errorf("search: embedding is empty")
	}
	// Placeholder: no-op until Qdrant is wired in.
	return nil
}
