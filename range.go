package pmtilr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
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
	// ReadRange reads the bytes defined by the Ranger and returns them,
	// or an error if reading fails.
	ReadRange(ctx context.Context, ranger Ranger) ([]byte, error)
}

// NewRangeReader parses a URI and returns an appropriate RangeReader implementation.
// Supports local file URIs ("file://") and bare paths. Other schemes are not supported.
func NewRangeReader(uri string) (RangeReader, error) {
	u, err := ParseURI(uri)
	if err != nil {
		return nil, fmt.Errorf("parsing URI %q: %w", uri, err)
	}

	switch u.Scheme() {
	case "", "file":
		return NewFileRangeReader(u.FullPath())
	case "s3":
		client := createS3Client()
		bucket, key := u.Host(), u.Path()
		return NewS3RangeReader(bucket, key, client)
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
// It validates the Ranger and handles io.EOF gracefully.
func (f *FileRangeReader) ReadRange(ctx context.Context, ranger Ranger) ([]byte, error) {
	if err := ranger.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ranger: %w", err)
	}

	offset := int64(ranger.Offset()) //nolint:gosec
	length := ranger.Length()

	// TODO: use sync.Pool maybe?
	// only issue is that buf size has high variance due to it
	// a) reading directories (large) and entries (small).
	// so sync.Pool needs to be big enough to avoid resizing, and small enough to not bloat
	// would also mean that the caller would be responsible to put back the buffer to the pool
	// which is meh.
	buf := make([]byte, length)

	// ReadAt may return io.EOF or io.ErrUnexpectedEOF, which are safe to ignore
	// but we clamp them to the actual returned data.
	n, err := f.file.ReadAt(buf, offset)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return buf[:n], nil
		}

		return nil, err
	}

	return buf, nil
}

// S3Client is an interface providing methods used by the S3RangeReader.
type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func createS3Client() S3Client {
	config := aws.NewConfig()
	return s3.NewFromConfig(*config)
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
// It validates the Ranger.
func (s *S3RangeReader) ReadRange(ctx context.Context, ranger Ranger) ([]byte, error) {
	if err := ranger.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ranger: %w", err)
	}

	offset := ranger.Offset()
	length := ranger.Length()

	// Define the byte range.
	byteRange := fmt.Sprintf("bytes=%d-%d", offset, offset+length-1)

	// Get object range.
	output, err := s.client.GetObject(ctx,
		&s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(s.key),
			Range:  aws.String(byteRange),
		},
		disableResponseValidation,
	)
	if err != nil {
		return []byte{}, err
	}
	defer output.Body.Close() //nolint:errcheck

	// Read the data into the buffer.
	buf := make([]byte, length)
	n, err := io.ReadFull(output.Body, buf)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			// Stream ended early, return partial content
			return buf[:n], nil
		}
		return nil, err
	}

	return buf, nil
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
