package buildinfo

import (
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"
)

var (
	buildInfo      *debug.BuildInfo
	buildInfoValid bool
	readBuildInfo  sync.Once

	externalURL     string
	readExternalURL sync.Once

	version     string
	readVersion sync.Once

	// Injected with ldflags at build!
	tag string
)

const (
	// develPrefix is prefixed to developer versions of the application.
	develPrefix = "v0.0.0-devel"
)

// Version returns the semantic version of the build.
// Use golang.org/x/mod/semver to compare versions.
func Version() string {
	readVersion.Do(func() {
		revision, valid := revision()
		if valid {
			revision = "+" + revision[:7]
		}
		if tag == "" {
			// This occurs when the tag hasn't been injected,
			// like when using "go run".
			version = develPrefix + revision
			return
		}
		version = "v" + tag
		// The tag must be prefixed with "v" otherwise the
		// semver library will return an empty string.
		if semver.Build(version) == "" {
			version += revision
		}
	})
	return version
}

// VersionsMatch compares the two versions. It assumes the versions match if
// the major and the minor versions are equivalent. Patch versions are
// disregarded. If it detects that either version is a developer build it
// returns true.
func VersionsMatch(v1, v2 string) bool {
	// Developer versions are disregarded...hopefully they know what they are
	// doing.
	if strings.HasPrefix(v1, develPrefix) || strings.HasPrefix(v2, develPrefix) {
		return true
	}

	v1Toks := strings.Split(v1, ".")
	v2Toks := strings.Split(v2, ".")

	// Versions should be formatted as "<major>.<minor>.<patch>".
	// We assume malformed versions are evidence of a bug and return false.
	if len(v1Toks) < 3 || len(v2Toks) < 3 {
		return false
	}

	// Slice off the patch suffix. Patch versions should be non-breaking
	// changes.
	v1MajorMinor := strings.Join(v1Toks[:2], ".")
	v2MajorMinor := strings.Join(v2Toks[:2], ".")

	return v1MajorMinor == v2MajorMinor
}

// ExternalURL returns a URL referencing the current Coder version.
// For production builds, this will link directly to a release.
// For development builds, this will link to a commit.
func ExternalURL() string {
	readExternalURL.Do(func() {
		repo := "https://github.com/coder/coder"
		revision, valid := revision()
		if !valid {
			externalURL = repo
			return
		}
		externalURL = fmt.Sprintf("%s/commit/%s", repo, revision)
	})
	return externalURL
}

// Time returns when the Git revision was published.
func Time() (time.Time, bool) {
	value, valid := find("vcs.time")
	if !valid {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic("couldn't parse time: " + err.Error())
	}
	return parsed, true
}

// revision returns the Git hash of the build.
func revision() (string, bool) {
	return find("vcs.revision")
}

// find panics if a setting with the specific key was not
// found in the build info.
func find(key string) (string, bool) {
	readBuildInfo.Do(func() {
		buildInfo, buildInfoValid = debug.ReadBuildInfo()
	})
	if !buildInfoValid {
		panic("couldn't read build info")
	}
	for _, setting := range buildInfo.Settings {
		if setting.Key != key {
			continue
		}
		return setting.Value, true
	}
	return "", false
}
