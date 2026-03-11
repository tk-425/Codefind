package chunker

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

type Chunk struct {
	Content    string
	StartLine  int
	EndLine    int
	Hash       string
	TokenCount int
}

type ChunkConfig struct {
	TargetTokens  int
	OverlapTokens int
	CharsPerToken float32
}

func DefaultConfig() ChunkConfig {
	return ChunkConfig{
		TargetTokens:  300,
		OverlapTokens: 50,
		CharsPerToken: 4.0,
	}
}

type WindowChunker struct {
	config ChunkConfig
}

func NewWindowChunker(config ChunkConfig) *WindowChunker {
	return &WindowChunker{config: config}
}

func (wc *WindowChunker) ChunkFile(content string, _ string) ([]Chunk, error) {
	chunks := []Chunk{}

	windowSize := int(float32(wc.config.TargetTokens) * wc.config.CharsPerToken)
	overlapSize := int(float32(wc.config.OverlapTokens) * wc.config.CharsPerToken)
	currentPos := 0
	currentLine := 1

	for currentPos < len(content) {
		chunkEnd := min(currentPos+windowSize, len(content))
		chunkText := content[currentPos:chunkEnd]
		linesInChunk := countLinesInText(chunkText)
		endLine := currentLine + linesInChunk - 1

		chunks = append(chunks, Chunk{
			Content:    chunkText,
			StartLine:  currentLine,
			EndLine:    endLine,
			Hash:       hashContent(chunkText),
			TokenCount: int(float32(len(chunkText)) / wc.config.CharsPerToken),
		})

		if chunkEnd == len(content) {
			break
		}

		currentPos += windowSize - overlapSize
		currentLine = endLine
	}

	return chunks, nil
}

func countLinesInText(text string) int {
	return strings.Count(text, "\n") + 1
}

func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}

func GenerateChunkID(repoID, filePath string, startLine, endLine int, content string) string {
	input := fmt.Sprintf("%s:%s:%d-%d:%s", repoID, filePath, startLine, endLine, content)
	hash := sha256.Sum256([]byte(input))
	uuidBytes := hash[:16]
	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x50
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		uuidBytes[0:4],
		uuidBytes[4:6],
		uuidBytes[6:8],
		uuidBytes[8:10],
		uuidBytes[10:16],
	)
}

func GenerateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}
