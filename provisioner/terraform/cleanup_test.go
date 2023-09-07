package terraform_test

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/slogtest"
	"github.com/coder/coder/v2/provisioner/terraform"
	"github.com/coder/coder/v2/testutil"
)

const cachePath = "/tmp/coder/provisioner-0/tf"

// updateGoldenFiles is a flag that can be set to update golden files.
var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

func TestPluginCache_Golden(t *testing.T) {
	t.Parallel()

	prepare := func() (afero.Fs, time.Time, slog.Logger) {
		fs := afero.NewMemMapFs()
		now := time.Date(2023, time.June, 3, 4, 5, 6, 0, time.UTC)
		logger := slogtest.Make(t, nil).
			Leveled(slog.LevelDebug).
			Named("cleanup-test")
		return fs, now, logger
	}

	t.Run("all plugins are stale", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
		defer cancel()

		fs, now, logger := prepare()

		// given
		// This plugin is older than 30 days.
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "terraform-provider-coder_v0.11.1", now.Add(-63*24*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "LICENSE", now.Add(-33*24*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "README.md", now.Add(-31*24*time.Hour))
		addPluginFolder(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "new_folder", now.Add(-31*24*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "new_folder/foobar.tf", now.Add(-43*24*time.Hour))

		// This plugin is older than 30 days.
		addPluginFile(t, fs, "registry.terraform.io/kreuzwerker/docker/2.25.0/darwin_arm64", "terraform-provider-docker_v2.25.0", now.Add(-31*24*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/kreuzwerker/docker/2.25.0/darwin_arm64", "LICENSE", now.Add(-32*24*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/kreuzwerker/docker/2.25.0/darwin_arm64", "README.md", now.Add(-33*24*time.Hour))

		// when
		terraform.CleanStaleTerraformPlugins(ctx, cachePath, fs, now, logger)

		// then
		diffFileSystem(t, fs)
	})

	t.Run("one plugin is stale", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
		defer cancel()

		fs, now, logger := prepare()

		// given
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "terraform-provider-coder_v0.11.1", now.Add(-2*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "LICENSE", now.Add(-3*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "README.md", now.Add(-4*time.Hour))
		addPluginFolder(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "new_folder", now.Add(-5*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "new_folder/foobar.tf", now.Add(-4*time.Hour))

		// This plugin is older than 30 days.
		addPluginFile(t, fs, "registry.terraform.io/kreuzwerker/docker/2.25.0/darwin_arm64", "terraform-provider-docker_v2.25.0", now.Add(-31*24*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/kreuzwerker/docker/2.25.0/darwin_arm64", "LICENSE", now.Add(-32*24*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/kreuzwerker/docker/2.25.0/darwin_arm64", "README.md", now.Add(-33*24*time.Hour))

		// when
		terraform.CleanStaleTerraformPlugins(ctx, cachePath, fs, now, logger)

		// then
		diffFileSystem(t, fs)
	})

	t.Run("one plugin file is touched", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), testutil.WaitShort)
		defer cancel()

		fs, now, logger := prepare()

		// given
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "terraform-provider-coder_v0.11.1", now.Add(-63*24*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "LICENSE", now.Add(-33*24*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "README.md", now.Add(-31*24*time.Hour))
		addPluginFolder(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "new_folder", now.Add(-4*time.Hour)) // touched
		addPluginFile(t, fs, "registry.terraform.io/coder/coder/0.11.1/darwin_arm64", "new_folder/foobar.tf", now.Add(-43*24*time.Hour))

		addPluginFile(t, fs, "registry.terraform.io/kreuzwerker/docker/2.25.0/darwin_arm64", "terraform-provider-docker_v2.25.0", now.Add(-31*24*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/kreuzwerker/docker/2.25.0/darwin_arm64", "LICENSE", now.Add(-2*time.Hour))
		addPluginFile(t, fs, "registry.terraform.io/kreuzwerker/docker/2.25.0/darwin_arm64", "README.md", now.Add(-33*24*time.Hour))

		// when
		terraform.CleanStaleTerraformPlugins(ctx, cachePath, fs, now, logger)

		// then
		diffFileSystem(t, fs)
	})
}

func addPluginFile(t *testing.T, fs afero.Fs, pluginPath string, resourcePath string, accessTime time.Time) {
	err := fs.MkdirAll(filepath.Join(cachePath, pluginPath), 0o755)
	require.NoError(t, err, "can't create test folder for plugin file")

	err = fs.Chtimes(filepath.Join(cachePath, pluginPath), accessTime, accessTime)
	require.NoError(t, err, "can't set times")

	err = afero.WriteFile(fs, filepath.Join(cachePath, pluginPath, resourcePath), []byte("foo"), 0o644)
	require.NoError(t, err, "can't create test file")

	err = fs.Chtimes(filepath.Join(cachePath, pluginPath, resourcePath), accessTime, accessTime)
	require.NoError(t, err, "can't set times")
}

func addPluginFolder(t *testing.T, fs afero.Fs, pluginPath string, folderPath string, accessTime time.Time) {
	err := fs.MkdirAll(filepath.Join(cachePath, pluginPath, folderPath), 0o755)
	require.NoError(t, err, "can't create plugin folder")

	err = fs.Chtimes(filepath.Join(cachePath, pluginPath, folderPath), accessTime, accessTime)
	require.NoError(t, err, "can't set times")
}

func diffFileSystem(t *testing.T, fs afero.Fs) {
	actual := dumpFileSystem(t, fs)

	partialName := strings.Join(strings.Split(t.Name(), "/")[1:], "_")
	goldenFile := filepath.Join("testdata", "cleanup-stale-plugins", partialName+".txt.golden")
	if *updateGoldenFiles {
		err := os.MkdirAll(filepath.Dir(goldenFile), 0o755)
		require.NoError(t, err, "want no error creating golden file directory")

		err = os.WriteFile(goldenFile, actual, 0o600)
		require.NoError(t, err, "want no error creating golden file")
		return
	}

	want, err := os.ReadFile(goldenFile)
	want = bytes.ReplaceAll(want, []byte{'\r', '\n'}, []byte{'\n'}) // fix for Windows
	require.NoError(t, err, "open golden file, run \"make update-golden-files\" and commit the changes")
	assert.Empty(t, cmp.Diff(want, actual), "golden file mismatch (-want +got): %s, run \"make update-golden-files\", verify and commit the changes", goldenFile)
}

func dumpFileSystem(t *testing.T, fs afero.Fs) []byte {
	var buffer bytes.Buffer
	err := afero.Walk(fs, "/", func(path string, info os.FileInfo, err error) error {
		_, _ = buffer.WriteString(path)
		_ = buffer.WriteByte(' ')
		if info.IsDir() {
			_ = buffer.WriteByte('d')
		} else {
			_ = buffer.WriteByte('f')
		}
		_ = buffer.WriteByte('\n')
		return nil
	})
	require.NoError(t, err, "can't dump the file system")
	return buffer.Bytes()
}
