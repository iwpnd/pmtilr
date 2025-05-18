package pmtilr

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemeString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		s        Scheme
		expected string
	}{
		{name: "file scheme", s: FileScheme, expected: "file"},
		{name: "s3 scheme", s: S3Scheme, expected: "s3"},
		{name: "unknown scheme", s: UnknownScheme, expected: "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.s.String(); got != tc.expected {
				t.Errorf("Scheme.String() = %q; expected %q", got, tc.expected)
			}
		})
	}
}

func TestParseURI(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		input             string
		expectedHost      string
		expectedPath      string
		expectedFullPath  string
		expectedScheme    string
		expectErr         bool
		expectErrContains string
	}{
		{
			name:             "no schema, empty input",
			input:            "",
			expectedHost:     "",
			expectedPath:     ".",
			expectedFullPath: ".",
			expectedScheme:   "file",
			expectErr:        false,
		},
		{
			name:             "no schema, absolute filepath",
			input:            "/home/user/data.csv",
			expectedHost:     "",
			expectedPath:     "/home/user/data.csv",
			expectedFullPath: "/home/user/data.csv",
			expectedScheme:   "file",
			expectErr:        false,
		},
		{
			name:             "no schema, relative filepath",
			input:            "./home/user/data.csv",
			expectedHost:     "",
			expectedPath:     "./home/user/data.csv",
			expectedFullPath: "home/user/data.csv",
			expectedScheme:   "file",
			expectErr:        false,
		},
		{
			name:             "no schema, relative filepath without dot notation",
			input:            "home/user/data.csv",
			expectedHost:     "",
			expectedPath:     "home/user/data.csv",
			expectedFullPath: "home/user/data.csv",
			expectedScheme:   "file",
			expectErr:        false,
		},
		{
			name:             "file schema, relative filepath",
			input:            "file://path/to/file.txt",
			expectedHost:     "path",
			expectedPath:     "/to/file.txt",
			expectedFullPath: "path/to/file.txt",
			expectedScheme:   "file",
			expectErr:        false,
		},
		{
			name:             "s3 schema",
			input:            "s3://mybucket/folder/object.txt",
			expectedHost:     "mybucket",
			expectedPath:     "/folder/object.txt",
			expectedFullPath: "mybucket/folder/object.txt",
			expectedScheme:   "s3",
			expectErr:        false,
		},
		{
			name:             "whitespaced uri string",
			input:            "   s3://bucket/key  ",
			expectedHost:     "bucket",
			expectedPath:     "/key",
			expectedFullPath: "bucket/key",
			expectedScheme:   "s3",
			expectErr:        false,
		},
		{
			name:              "unsupported scheme",
			input:             "ftp://example.com/resource",
			expectErr:         true,
			expectErrContains: "unsupported URI scheme",
		},
		{
			name:              "invalid uri",
			input:             "http://%",
			expectErr:         true,
			expectErrContains: "parsing URI",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			u, err := ParseURI(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("ParseURI(%q) expected error, got nil", tc.input)
				}
				if tc.expectErrContains != "" &&
					!strings.Contains(err.Error(), tc.expectErrContains) {
					t.Errorf("error %v does not contain %q", err, tc.expectErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseURI(%q) unexpected error: %v", tc.input, err)
			}

			if got := u.Host(); got != tc.expectedHost {
				t.Errorf("Host() = %q; expected %q", got, tc.expectedHost)
			}
			if got := u.Path(); got != tc.expectedPath {
				t.Errorf("Path() = %q; expected %q", got, tc.expectedPath)
			}

			// Normalize for comparison
			gotFull := filepath.ToSlash(u.FullPath())
			if gotFull != tc.expectedFullPath {
				t.Errorf("FullPath() = %q; expected %q", gotFull, tc.expectedFullPath)
			}

			if got := u.Scheme(); got != tc.expectedScheme {
				t.Errorf("Scheme() = %q; expected %q", got, tc.expectedScheme)
			}
		})
	}
}
