package content

import (
	"strings"
	"unicode"
)

func Slugify(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(input))
	previousDash := false

	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			previousDash = false
			continue
		}
		if !previousDash {
			b.WriteByte('-')
			previousDash = true
		}
	}

	return strings.Trim(b.String(), "-")
}
