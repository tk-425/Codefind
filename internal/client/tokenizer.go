package client

import (
	"crypto/md5"
	"fmt"
	"strings"
	"sync"

	"github.com/tk-425/Codefind/internal/chunker"
)

// Tokenizer wraps chunks and tokenizes them via server
type Tokenizer struct {
	client *APIClient
	model  string
	cache  map[string]int // Cache: content hash -> token count
	mu     sync.Mutex
}

// NewTokenizer creates a new tokenizer
func NewTokenizer(client *APIClient, model string) *Tokenizer {
	return &Tokenizer{
		client: client,
		model:  model,
		cache:  make(map[string]int),
	}
}

// TokenizeChunks tokenizes chunks and returns verified chunks
func (t *Tokenizer) TokenizeChunks(chunks []chunker.Chunk, maxTokens int) ([]chunker.Chunk, error) {
	verified := []chunker.Chunk{}
	batchSize := 16

	for i := 0; i < len(chunks); i += batchSize {
		end := min(i+batchSize, len(chunks))

		batch := chunks[i:end]
		if err := t.processBatch(batch, &verified, maxTokens); err != nil {
			return nil, err
		}
	}

	return verified, nil
}

// processBatch tokenizes a batch of chunks and re-splits over-limit chunks
func (t *Tokenizer) processBatch(chunks []chunker.Chunk, verified *[]chunker.Chunk, maxTokens int) error {
	texts := []string{}
	indices := []int{}

	for i, chunk := range chunks {
		// Check cache first
		t.mu.Lock()
		if tokenCount, cached := t.cache[chunk.Hash]; cached {
			t.mu.Unlock()
			chunk.TokenCount = tokenCount

			// If cached chunk exceeds limit, split it
			if tokenCount > maxTokens {
				if err := t.splitAndProcess(chunk, verified, maxTokens); err != nil {
					return err
				}
			} else {
				*verified = append(*verified, chunk)
			}
			continue
		}
		t.mu.Unlock()

		// Add to batch for tokenization
		texts = append(texts, chunk.Content)
		indices = append(indices, i)
	}

	// If all chunks were cached, return early
	if len(texts) == 0 {
		return nil
	}

	// Tokenize uncached chunks
	tokens, err := t.client.Tokenize(t.model, texts)
	if err != nil {
		return fmt.Errorf("tokenization failed: %w", err)
	}

	// Update chunks with token counts and handle over-limit chunks
	for batchIdx, originalIdx := range indices {
		chunk := chunks[originalIdx]
		tokenCount := tokens[batchIdx]

		chunk.TokenCount = tokenCount

		// Cache the result
		t.mu.Lock()
		t.cache[chunk.Hash] = tokenCount
		t.mu.Unlock()

		if tokenCount > maxTokens {
			// Chunk exceeds limit, split it
			if err := t.splitAndProcess(chunk, verified, maxTokens); err != nil {
				return err
			}
		} else {
			*verified = append(*verified, chunk)
		}
	}

	return nil
}

// CacheSize returns the number of cached items
func (t *Tokenizer) CacheSize() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.cache)
}

// ClearCache clears the tokenization cache
func (t *Tokenizer) ClearCache() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cache = make(map[string]int)
}

// splitAndProcess splits an over-limit chunk recursively until all parts are under maxTokens
func (t *Tokenizer) splitAndProcess(chunk chunker.Chunk, verified *[]chunker.Chunk, maxTokens int) error {
	// Calculate window size based on actual token count
	// Estimate chars per token in this chunk
	charsPerToken := float32(len(chunk.Content)) / float32(chunk.TokenCount)

	// Use 90% of maxTokens as target to avoid edge cases
	windowSize := int(float32(maxTokens) * charsPerToken * 0.9)

	if windowSize < 50 {
		windowSize = 50 // Minimum to avoid infinite recursion
	}

	overlapSize := windowSize / 10 // 10% overlap

	content := chunk.Content
	currentPos := 0
	currentLine := chunk.StartLine

	for currentPos < len(content) {
		chunkEnd := currentPos + windowSize
		if chunkEnd > len(content) {
			chunkEnd = len(content)
		}

		chunkText := content[currentPos:chunkEnd]

		// Count newlines to calculate end line
		newlineCount := strings.Count(chunkText, "\n")
		endLine := currentLine + newlineCount

		subChunk := chunker.Chunk{
			Content:   chunkText,
			StartLine: currentLine,
			EndLine:   endLine,
			Hash:      hashContent(chunkText),
		}

		// Tokenize the sub-chunk
		tokens, err := t.client.Tokenize(t.model, []string{chunkText})
		if err != nil {
			return fmt.Errorf("tokenization of split chunk failed: %w", err)
		}

		subChunk.TokenCount = tokens[0]

		// Cache the result
		t.mu.Lock()
		t.cache[subChunk.Hash] = tokens[0]
		t.mu.Unlock()

		if tokens[0] > maxTokens {
			// Still over limit, recurse
			if err := t.splitAndProcess(subChunk, verified, maxTokens); err != nil {
				return err
			}
		} else {
			*verified = append(*verified, subChunk)
		}

		// Move to next window with overlap
		currentPos += windowSize - overlapSize
		currentLine = endLine

		if chunkEnd == len(content) {
			break
		}
	}

	return nil
}

// hashContent returns MD5 hash of content
func hashContent(content string) string {
	h := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", h)
}
