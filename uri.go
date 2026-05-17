package pmtilr

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type Scheme uint8

const (
	SchemeUnknown Scheme = iota
	SchemeFile
	SchemeS3
	SchemeHTTP
	SchemeHTTPS
	SchemeFileCwd
)

var _ fmt.Stringer = SchemeUnknown

var schemeStrings = map[Scheme]string{
	SchemeFile:    "file",
	SchemeS3:      "s3",
	SchemeHTTP:    "http",
	SchemeHTTPS:   "https",
	SchemeUnknown: "unknown",
	SchemeFileCwd: "",
}

func (s Scheme) String() string {
	return schemeStrings[s]
}

// URI encapsulates parsed URI components.
type URI struct {
	raw      *url.URL
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

func (u *URI) Scheme() Scheme {
	return u.scheme
}

func (u *URI) Raw() *url.URL {
	return u.raw
}

func newURI(u *url.URL, scheme Scheme) *URI {
	p := filepath.FromSlash(filepath.Join(u.Host, u.Path))
	return &URI{
		raw:      u,
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
		// empty input / SchemeFileCwd -> treat as file at cwd
		return newURI(&url.URL{Path: "."}, SchemeFile), nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing URI %q: %w", raw, err)
	}

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case SchemeHTTP.String(), SchemeHTTPS.String():
		return newURI(u, SchemeHTTP), nil
	case SchemeFileCwd.String(), SchemeFile.String():
		return newURI(u, SchemeFile), nil
	case SchemeS3.String():
		return newURI(u, SchemeS3), nil
	default:
		return nil, fmt.Errorf("unsupported URI scheme %q", u.Scheme)
	}
}
