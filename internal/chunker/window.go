package chunker

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"strings"
)

// Chunk represents a code chunk with metadata
type Chunk struct {
	Content    string
	StartLine  int
	EndLine    int
	Hash       string // MD5 hash of content
	TokenCount int    // Token count (verified by server)
}

// ChunkConfig contains chunking parameters
type ChunkConfig struct {
	TargetTokens  int     // Target tokens per chunk (default: 450)
	OverlapTokens int     // Overlap between chunks (default: 50)
	CharsPerToken float32 // Characters per token estimate (default: 4.0)
}

// DefaultConfig returns default chunking configuration
func DefaultConfig() ChunkConfig {
	return ChunkConfig{
		TargetTokens:  300, // Conservative target to stay well under BERT's 512 max
		OverlapTokens: 50,
		CharsPerToken: 4.0,
	}
}

// WindowChunker chunks text using sliding window approach
type WindowChunker struct {
	config ChunkConfig
}

// NewWindowChunker creates a new window chunker
func NewWindowChunker(config ChunkConfig) *WindowChunker {
	return &WindowChunker{config: config}
}

// ChunkFile splits file content into chunks
func (wc *WindowChunker) ChunkFile(content string, filePath string) ([]Chunk, error) {
	chunks := []Chunk{}

	windowSize := int(float32(wc.config.TargetTokens) * wc.config.CharsPerToken)
	overlapSize := int(float32(wc.config.OverlapTokens) * wc.config.CharsPerToken)

	currentPos := 0
	currentLine := 1

	for currentPos < len(content) {
		// Calculate chunk end
		chunkEnd := min(currentPos+windowSize, len(content))

		// Extract chunk text
		chunkText := content[currentPos:chunkEnd]

		// Count lines in chunk and calculate end line
		linesInChunk := countLinesInText(chunkText)
		endLine := currentLine + linesInChunk - 1

		// Create chunk
		chunk := Chunk{
			Content:    chunkText,
			StartLine:  currentLine,
			EndLine:    endLine,
			Hash:       hashContent(chunkText),
			TokenCount: int(float32(len(chunkText)) / wc.config.CharsPerToken),
		}

		chunks = append(chunks, chunk)

		// Move position with overlap
		currentPos += windowSize - overlapSize
		currentLine = endLine

		// Stop if we've reached the end
		if chunkEnd == len(content) {
			break
		}
	}

	return chunks, nil
}

// countLinesInText counts newline characters in text
func countLinesInText(text string) int {
	return strings.Count(text, "\n") + 1
}

// hashContent returns MD5 hash of content
func hashContent(content string) string {
	h := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", h)
}

// GenerateChunkID creates a stable ID based on content and location
// This ensures the same content at the same location always gets the same ID
func GenerateChunkID(repoID, filePath string, startLine, endLine int, content string) string {
	// Combine identifying factors
	input := fmt.Sprintf("%s:%s:%d-%d:%s", repoID, filePath, startLine, endLine, content)

	// Hash to fixed length
	hash := sha256.Sum256([]byte(input))

	// Return first 16 chars of hex (64-bit uniqueness)
	return fmt.Sprintf("%x", hash[:8])
}

// GenerateContentHash creates SHA256 hash of content
func GenerateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}
