package terraform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
)

// nolint:paralleltest
func Test_absoluteBinaryPath(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name             string
		args             args
		terraformVersion string
		expectedErr      error
	}{
		{
			name:             "TestCorrectVersion",
			args:             args{ctx: context.Background()},
			terraformVersion: "1.1.9",
			expectedErr:      nil,
		},
		{
			name:             "TestOldVersion",
			args:             args{ctx: context.Background()},
			terraformVersion: "1.0.9",
			expectedErr:      xerrors.Errorf("Terraform binary minor version mismatch."),
		},
		{
			name:             "TestNewVersion",
			args:             args{ctx: context.Background()},
			terraformVersion: "1.2.9",
			expectedErr:      xerrors.Errorf("Terraform binary minor version mismatch."),
		},
		{
			name:             "TestMalformedVersion",
			args:             args{ctx: context.Background()},
			terraformVersion: "version",
			expectedErr:      xerrors.Errorf("Terraform binary get version failed: Malformed version: version"),
		},
	}
	// nolint:paralleltest
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if runtime.GOOS == "windows" {
				t.Skip("Dummy terraform executable on Windows requires sh which isn't very practical.")
			}

			// Create a temp dir with the binary
			tempDir := t.TempDir()
			terraformBinaryOutput := fmt.Sprintf(`#!/bin/sh
			cat <<-EOF
			{
				"terraform_version": "%s",
				"platform": "linux_amd64",
				"provider_selections": {},
				"terraform_outdated": false
			}
			EOF`, tt.terraformVersion)

			// #nosec
			err := os.WriteFile(
				filepath.Join(tempDir, "terraform"),
				[]byte(terraformBinaryOutput),
				0770,
			)
			require.NoError(t, err)

			// Add the binary to PATH
			pathVariable := os.Getenv("PATH")
			t.Setenv("PATH", strings.Join([]string{tempDir, pathVariable}, ":"))

			var expectedAbsoluteBinary string
			if tt.expectedErr == nil {
				expectedAbsoluteBinary = filepath.Join(tempDir, "terraform")
			}

			actualAbsoluteBinary, actualErr := absoluteBinaryPath(tt.args.ctx)
			if actualAbsoluteBinary != expectedAbsoluteBinary {
				t.Errorf("getAbsoluteBinaryPath() absoluteBinaryPath, actual = %v, expected %v", actualAbsoluteBinary, expectedAbsoluteBinary)
			}
			if tt.expectedErr == nil {
				if actualErr != nil {
					t.Errorf("getAbsoluteBinaryPath() error, actual = %v, expected %v", actualErr.Error(), tt.expectedErr)
				}
			} else {
				if actualErr.Error() != tt.expectedErr.Error() {
					t.Errorf("getAbsoluteBinaryPath() error, actual = %v, expected %v", actualErr.Error(), tt.expectedErr.Error())
				}
			}
		})
	}
}
