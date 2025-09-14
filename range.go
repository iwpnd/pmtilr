package pmtilr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	indexOffset  = 0
	indexSize    = 1
	indexMinZoom = 0
	indexMaxZoom = 1
)

// ZoomRange defines an inclusive range of zoom levels [MinZoom, MaxZoom].
// It supports validation to ensure MinZoom <= MaxZoom.
type ZoomRange [2]uint64

// Validate ensures that the minimum zoom is not greater than the maximum.
// Returns an error if the range is invalid.
func (zr ZoomRange) Validate() error {
	minZoom, maxZoom := zr[0], zr[1]
	if minZoom > maxZoom {
		return fmt.Errorf("min zoom %d cannot be greater than max zoom %d", minZoom, maxZoom)
	}
	return nil
}

// MinZoom returns the lower bound of the zoom range.
func (zr ZoomRange) MinZoom() uint64 {
	return zr[indexMinZoom]
}

// MaxZoom returns the upper bound of the zoom range.
func (zr ZoomRange) MaxZoom() uint64 {
	return zr[indexMaxZoom]
}

// NewZoomRange constructs a ZoomRange with the specified minimum and maximum zoom.
func NewZoomRange(minZoom, maxZoom uint64) ZoomRange {
	return ZoomRange{minZoom, maxZoom}
}

// Ranger combines Offsetter and Sizer, adding a Validate method
// to ensure the range parameters are valid.
type Ranger interface {
	Offset() uint64
	Length() uint64
	Validate() error
}

// Range represents a byte range with an Offset and a Size.
// It implements the Ranger interface.
type Range [2]uint64

// Offset returns the starting byte offset of the range.
func (r Range) Offset() uint64 {
	return r[indexOffset]
}

// Length returns the number of bytes to read in the range.
func (r Range) Length() uint64 {
	return r[indexSize]
}

// Validate ensures that the range size is positive.
// Returns an error if Size() == 0.
func (r Range) Validate() error {
	if r.Length() == 0 {
		return errors.New("invalid range: size must be a positive integer")
	}
	return nil
}

// NewRange constructs a Range with the given offset and size.
func NewRange(offset, size uint64) Range {
	return Range{offset, size}
}

// RangeReader defines the interface for reading arbitrary byte ranges
// given a Ranger description.
type RangeReader interface {
	// ReadRange reads the bytes defined by the Ranger and returns a ReadCloser,
	// or an error if reading fails. The caller is responsible for closing the ReadCloser.
	ReadRange(ctx context.Context, ranger Ranger) (io.ReadCloser, error)
}

// NewRangeReader parses a URI and returns an appropriate RangeReader implementation.
// Supports local file URIs ("file://") and bare paths. Other schemes are not supported.
func NewRangeReader(ctx context.Context, uri string) (RangeReader, error) {
	u, err := ParseURI(uri)
	if err != nil {
		return nil, fmt.Errorf("parsing URI %q: %w", uri, err)
	}

	switch u.Scheme() {
	case "", "file":
		return NewFileRangeReader(u.FullPath())
	case "s3":
		client, err := createS3Client(ctx)
		if err != nil {
			return nil, err
		}
		bucket, key := u.Host(), u.Path()
		return NewS3RangeReader(bucket, strings.TrimPrefix(key, "/"), client)
	default:
		return nil, fmt.Errorf("unsupported URI scheme %q", u.Scheme())
	}
}

// FileRangeReader implements RangeReader by reading from an io.ReaderAt (file).
// It interprets Ranger.Offset() and Ranger.Size() to slice the file.
type FileRangeReader struct {
	file io.ReaderAt
}

// NewFileRangeReader opens the file at the given path and returns a FileRangeReader.
func NewFileRangeReader(path string) (*FileRangeReader, error) {
	filePath := filepath.Clean(path)
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file at path %s: %w", path, err)
	}
	return &FileRangeReader{file: f}, nil
}

// ReadRange reads bytes from the underlying file at the specified range.
// It validates the Ranger and returns a ReadCloser using SectionReader for streaming access.
func (f *FileRangeReader) ReadRange(ctx context.Context, ranger Ranger) (io.ReadCloser, error) {
	if err := ranger.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ranger: %w", err)
	}
	return io.NopCloser(
		io.NewSectionReader(
			f.file, int64(ranger.Offset()), int64(ranger.Length()), //nolint:gosec
		),
	), nil
}

// S3Client is an interface providing methods used by the S3RangeReader.
type S3Client interface {
	GetObject(
		ctx context.Context,
		params *s3.GetObjectInput,
		optFns ...func(*s3.Options),
	) (*s3.GetObjectOutput, error)
}

func createS3Client(ctx context.Context) (S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	}), nil
}

// S3RangeReader implements RangeReader by reading from an S3 bucket
type S3RangeReader struct {
	client S3Client
	bucket string
	key    string
}

// NewS3RangeReader creates a S3RangeReader implementing RangeReader.
func NewS3RangeReader(bucket, key string, client S3Client) (*S3RangeReader, error) {
	return &S3RangeReader{
		bucket: bucket,
		key:    key,
		client: client,
	}, nil
}

// ReadRange reads bytes from the underlying S3 object at the specified range.
// It validates the Ranger and returns a ReadCloser for streaming access.
func (s *S3RangeReader) ReadRange(ctx context.Context, ranger Ranger) (io.ReadCloser, error) {
	if err := ranger.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ranger: %w", err)
	}

	byteRange := bytesRange(ranger.Offset(), ranger.Length())
	output, err := s.client.GetObject(ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(s.key),
			Range:  aws.String(byteRange),
		},
		disableResponseValidation,
	)
	if err != nil {
		return nil, err
	}

	return output.Body, nil
}

// disableResponseValidation disables checksum validation on the response.  This
// is necessary for S3 ReaderAt byte range requests as the responses to these do
// not include checksums.  Not disabling checksums means that by default the AWS
// SDK will log checksum failures.  We *could* disable this logging using
// DisableLogOutputChecksumValidationSkipped but it seems cleaner to disable the
// check full stop.
func disableResponseValidation(o *s3.Options) {
	o.ResponseChecksumValidation = aws.ResponseChecksumValidationUnset
}

func bytesRange(offset, length uint64) string {
	bufPtr, _ := keyBufPool.Get().(*[]byte) //nolint:errcheck
	buf := (*bufPtr)[:0]                    // Reset length but keep capacity
	defer keyBufPool.Put(bufPtr)

	buf = append(buf, "bytes="...)
	buf = strconv.AppendUint(buf, offset, 10)
	buf = append(buf, '-')
	buf = strconv.AppendUint(buf, offset+length-1, 10)

	return string(buf)
}
