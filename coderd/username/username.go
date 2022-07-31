package username

import (
	"regexp"
	"strings"

	"github.com/moby/moby/pkg/namesgenerator"
)

var (
	valid   = regexp.MustCompile("^[a-zA-Z0-9]+(?:-[a-zA-Z0-9]+)*$")
	replace = regexp.MustCompile("[^a-zA-Z0-9-]*")
)

// Valid returns whether the input string is a valid username or not.
func Valid(str string) bool {
	if len(str) > 32 {
		return false
	}
	if len(str) < 1 {
		return false
	}
	return valid.MatchString(str)
}

// From returns a best-effort username from the provided string.
//
// It first attempts to validate the incoming string, which will
// be returned if it is valid. It then will attempt to extract
// the username from an email address. If no success happens during
// these steps, a random username will be returned.
func From(str string) string {
	if Valid(str) {
		return str
	}
	emailAt := strings.LastIndex(str, "@")
	if emailAt >= 0 {
		str = str[:emailAt]
	}
	str = replace.ReplaceAllString(str, "")
	if Valid(str) {
		return str
	}
	return strings.ReplaceAll(namesgenerator.GetRandomName(1), "_", "-")
}
