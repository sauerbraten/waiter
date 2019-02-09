package cubecode

import (
	"regexp"
	"strings"
	"unicode"
)

// Matches sauer color codes (sauer uses form feed followed by a digit, e.g. \f3 for red)
var sauerStringsSanitizer = regexp.MustCompile("\\f\\d")

// SanitizeString returns the string, cleared of sauer color codes like \f3 for red.
func SanitizeString(s string) string {
	s = sauerStringsSanitizer.ReplaceAllLiteralString(s, "")
	return strings.TrimSpace(s)
}

func Filter(s string, whitespaceAllowed bool) (filtered string) {
	s = SanitizeString(s)

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || (whitespaceAllowed && unicode.IsSpace(r)) {
			filtered += string(r)
		}
	}

	return
}
