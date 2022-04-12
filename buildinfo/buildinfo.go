package buildinfo

import (
	"path"
	"runtime/debug"
	"sync"
	"time"

	"golang.org/x/mod/semver"
)

var (
	buildInfo      *debug.BuildInfo
	buildInfoValid bool
	readBuildInfo  sync.Once

	// Injected with ldflags at build!
	tag string
)

// Version returns the semantic version of the build.
// Use golang.org/x/mod/semver to compare versions.
func Version() string {
	revision, valid := revision()
	if valid {
		revision = "+" + revision[:7]
	}
	if tag == "" {
		return "v0.0.0-devel" + revision
	}
	if semver.Build(tag) == "" {
		tag += revision
	}
	return "v" + tag
}

// ExternalURL returns a URL referencing the current Coder version.
// For production builds, this will link directly to a release.
// For development builds, this will link to a commit.
func ExternalURL() string {
	repo := "https://github.com/coder/coder"
	revision, valid := revision()
	if !valid {
		return repo
	}
	return path.Join(repo, "commit", revision)
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
