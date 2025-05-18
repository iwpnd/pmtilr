package pmtilr

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type Scheme uint8

const (
	UnknownScheme Scheme = iota
	FileScheme
	S3Scheme
)

var _ fmt.Stringer = UnknownScheme

var schemeStrings = map[Scheme]string{
	FileScheme:    "file",
	S3Scheme:      "s3",
	UnknownScheme: "unknown",
}

func (s Scheme) String() string {
	return schemeStrings[s]
}

// URI encapsulates parsed URI components.
type URI struct {
	host     string
	path     string
	fullPath string
	scheme   Scheme
}

func (u *URI) Host() string {
	return u.host
}

func (u *URI) Path() string {
	return u.path
}

func (u *URI) FullPath() string {
	return u.fullPath
}

func (u *URI) Scheme() string {
	return u.scheme.String()
}

func newURI(u *url.URL, scheme Scheme) *URI {
	p := filepath.FromSlash(filepath.Join(u.Host, u.Path))
	return &URI{
		host:     u.Host,
		path:     u.Path,
		fullPath: p,
		scheme:   scheme,
	}
}

// ParseURI parses a string into a URI struct, trimming whitespace and handling supported schemes.
func ParseURI(raw string) (*URI, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		// empty input -> treat as file at cwd
		return newURI(&url.URL{Path: "."}, FileScheme), nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing URI %q: %w", raw, err)
	}

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "", "file":
		return newURI(u, FileScheme), nil
	case "s3":
		return newURI(u, S3Scheme), nil
	default:
		return nil, fmt.Errorf("unsupported URI scheme %q", u.Scheme)
	}
}
