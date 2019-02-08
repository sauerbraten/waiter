package cubecode

import (
	"regexp"
	"strings"
)

// Matches sauer color codes (sauer uses form feed followed by a digit, e.g. \f3 for red)
var sauerStringsSanitizer = regexp.MustCompile("\\f.")

// SanitizeString returns the string, cleared of sauer color codes like \f3 for red.
func SanitizeString(s string) string {
	s = sauerStringsSanitizer.ReplaceAllLiteralString(s, "")
	return strings.TrimSpace(s)
}
