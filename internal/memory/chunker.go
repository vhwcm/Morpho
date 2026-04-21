package memory

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

func SplitIntoChunks(text string, maxChunkChars, overlapChars int) []string {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return nil
	}
	if maxChunkChars <= 0 {
		maxChunkChars = 700
	}
	if overlapChars < 0 {
		overlapChars = 0
	}
	if overlapChars >= maxChunkChars {
		overlapChars = maxChunkChars / 4
	}

	runes := []rune(clean)
	out := make([]string, 0)
	step := maxChunkChars - overlapChars
	for start := 0; start < len(runes); start += step {
		end := start + maxChunkChars
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			out = append(out, chunk)
		}
		if end == len(runes) {
			break
		}
	}
	return out
}

func HashChunk(text string) string {
	h := sha1.Sum([]byte(strings.TrimSpace(text)))
	return hex.EncodeToString(h[:])
}

func estimateTokens(text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	return len(strings.Fields(text))
}
