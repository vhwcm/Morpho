package memory

import (
	"regexp"
	"strings"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)gemini[_-]?api[_-]?key\s*[:=]\s*[\"']?([a-z0-9_\-]{10,})[\"']?`),
	regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*[\"']?([a-z0-9_\-]{10,})[\"']?`),
	regexp.MustCompile(`(?i)token\s*[:=]\s*[\"']?([a-z0-9_\-]{10,})[\"']?`),
}

func Sanitize(text string) string {
	out := text
	for _, re := range secretPatterns {
		out = re.ReplaceAllString(out, "[REDACTED_SECRET]")
	}
	return strings.TrimSpace(out)
}
