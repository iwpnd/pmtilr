package pmtilr_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/iwpnd/pmtilr"
)

func TestFileRangeReader(t *testing.T) {
	testFileName := "testfile"
	testData := []byte("This is some test data for the RangeReader implementation.")
	setupFn := func(t *testing.T) string {
		t.Helper()

		d := t.TempDir()
		file := filepath.Join(d, testFileName)
		err := os.WriteFile(file, testData, 0o600)
		if err != nil {
			t.Fatalf("writing testdata should not error")
		}
		return file
	}

	tests := []struct {
		name          string
		offset        int64
		length        int
		setupFn       func(t *testing.T) string
		expectedData  string
		expectedError error
	}{
		{
			name:          "Read middle range",
			offset:        5,
			length:        10,
			setupFn:       setupFn,
			expectedData:  "is some te",
			expectedError: nil,
		},
		{
			name:          "Read full range",
			offset:        0,
			length:        len(testData),
			setupFn:       setupFn,
			expectedData:  string(testData),
			expectedError: nil,
		},
		{
			name:          "Read beyond end",
			offset:        int64(len(testData) - 5),
			length:        50,
			setupFn:       setupFn,
			expectedData:  "tion.",
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := tt.setupFn(t)
			reader, err := pmtilr.NewFileRangeReader(testFile)
			if err != nil {
				t.Fatal("unexpected error")
			}

			result, err := reader.ReadRange(t.Context(), pmtilr.NewRange(uint64(tt.offset), uint64(tt.length)))
			if !errors.Is(err, tt.expectedError) {
				t.Fatal("expected error, and received error do not match")
			}

			if len(tt.expectedData) != len(result) {
				t.Fatalf("expected equal length of expected data %d and got data %d", len(tt.expectedData), len(result))
			}

			if tt.expectedData != string(result) {
				t.Fatalf("expected %s, got: %s", tt.expectedData, string(result))
			}
		})
	}
}

func TestS3RangeReader(t *testing.T) {
	bucketName := "test-bucket"
	objectKey := "test-object"
	testData := []byte("This is some test data for the RangeReader implementation.")

	tests := []struct {
		name          string
		offset        int64
		length        int
		expectedData  string
		expectedError error
	}{
		{
			name:          "Read middle range",
			offset:        5,
			length:        10,
			expectedData:  "is some te",
			expectedError: nil,
		},
		{
			name:          "Read full range",
			offset:        0,
			length:        len(testData),
			expectedData:  string(testData),
			expectedError: nil,
		},
		{
			name:          "Read beyond end",
			offset:        int64(len(testData) - 5),
			length:        50,
			expectedData:  "tion.",
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockS3Client{
				GetObjectFunc: func(_ context.Context, params *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
					if bucketName != aws.ToString(params.Bucket) {
						t.Fatalf("expected: %s, got: %s", bucketName, aws.ToString(params.Bucket))
					}

					var start, end int
					r := aws.ToString(params.Range)
					_, err := fmt.Sscanf(r, "bytes=%d-%d", &start, &end)
					if err != nil {
						return nil, fmt.Errorf("invalid range header: %w", err)
					}

					dataLength := len(testData)
					if start >= dataLength {
						return &s3.GetObjectOutput{
							Body: io.NopCloser(bytes.NewReader(nil)), // No data
						}, nil
					}

					// Clamp end to test data length
					if end >= dataLength {
						end = dataLength - 1
					}

					if end < start {
						return &s3.GetObjectOutput{
							Body: io.NopCloser(bytes.NewReader(nil)),
						}, nil
					}

					d := end - start + 1
					buf := make([]byte, d)
					copy(buf, testData[start:end+1])

					return &s3.GetObjectOutput{
						Body: io.NopCloser(bytes.NewReader(buf)),
					}, nil
				},
			}

			reader, err := pmtilr.NewS3RangeReader(bucketName, objectKey, mockClient)
			if err != nil {
				t.Fatal("unexpected error")
			}

			result, err := reader.ReadRange(t.Context(), pmtilr.NewRange(uint64(tt.offset), uint64(tt.length)))
			if !errors.Is(err, tt.expectedError) {
				t.Fatalf("expected error, and received error do not match")
			}

			if len(tt.expectedData) != len(result) {
				t.Fatalf("expected equal length of expected data %d and got data %d", len(tt.expectedData), len(result))
			}

			if tt.expectedData != string(result) {
				t.Fatalf("expected %s, got: %s", tt.expectedData, string(result))
			}
		})
	}
}

type mockS3Client struct {
	GetObjectFunc func(ctx context.Context, params *s3.GetObjectInput) (*s3.GetObjectOutput, error)
}

func (m *mockS3Client) GetObject(
	ctx context.Context,
	params *s3.GetObjectInput,
	_ ...func(*s3.Options),
) (*s3.GetObjectOutput, error) {
	return m.GetObjectFunc(ctx, params)
}
