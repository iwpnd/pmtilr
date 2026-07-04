package pmtilr

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"

	singleflight "github.com/iwpnd/singleflightx"
)

func TestEntriesDeserializeNilReceiver(t *testing.T) {
	var e Entries
	br := bufio.NewReader(bytes.NewReader(nil))

	err := e.deserialize(br)
	if err == nil || !strings.Contains(err.Error(), "cannot deserialize") {
		t.Errorf("expected nil slice error, got: %v", err)
	}
}

func TestReadEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		dataFunc      func() []byte
		expectErr     bool
		expectEntries []Entry
	}{
		{
			name: "valid multiple entries with offset propagation",
			dataFunc: func() []byte {
				// Two entries:
				// Entry 0: TileID = 3, RunLength = 2, Length = 100, Offset = 500 (actual = 499)
				// Entry 1: TileID delta = 1 (=> 4), RunLength = 1, Length = 50, Offset = 0 (should use offset = 499 + 100 = 599)
				buf := &bytes.Buffer{}
				writeUvarint(buf, 2) // count

				// TileID deltas
				writeUvarint(buf, 3) // delta1
				writeUvarint(buf, 1) // delta2

				// RunLengths
				writeUvarint(buf, 2)
				writeUvarint(buf, 1)

				// Lengths
				writeUvarint(buf, 100)
				writeUvarint(buf, 50)

				// Offsets (stored +1 in PMTiles, and 0 triggers propagation)
				writeUvarint(buf, 500) // actual offset = 499
				writeUvarint(buf, 0)   // triggers propagation

				return buf.Bytes()
			},
			expectErr: false,
			expectEntries: []Entry{
				{TileID: 3, RunLength: 2, Length: 100, Offset: 499},
				{TileID: 4, RunLength: 1, Length: 50, Offset: 599}, // offset from previous + length
			},
		},
		{
			name: "invalid truncated multi-entry",
			dataFunc: func() []byte {
				buf := &bytes.Buffer{}
				writeUvarint(buf, 2)
				writeUvarint(buf, 1)
				// missing remaining fields
				return buf.Bytes()
			},
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key := "1337:31337"
			mockReader := &mockRangeReader{
				data: map[string][]byte{
					key: tc.dataFunc(),
				},
			}

			readCloser, err := mockReader.ReadRange(t.Context(), mockRanger{1337, 31337})
			if err != nil {
				t.Fatalf("mockRangeReader failed: %v", err)
			}
			defer readCloser.Close()

			br := bufio.NewReader(readCloser)
			entries, err := readEntries(br)

			if tc.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(entries) != len(tc.expectEntries) {
				t.Fatalf("expected %d entries, got %d", len(tc.expectEntries), len(entries))
			}

			for i, want := range tc.expectEntries {
				if entries[i] != want {
					t.Errorf("entry[%d] mismatch:\n  got:  %+v\n  want: %+v", i, entries[i], want)
				}
			}
		})
	}
}

func TestRepositoryDirectoryAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		reader          *mockRangeReader
		header          HeaderV3
		ranger          mockRanger
		decompress      DecompressFunc
		expectError     bool
		expectFromCache bool
	}{
		{
			name: "success on cache miss",
			reader: &mockRangeReader{
				data: map[string][]byte{
					"1337:31337": generateFakeDirectoryData(10),
				},
			},
			header:          fakeHeader("etag1337"),
			ranger:          mockRanger{1337, 31337},
			decompress:      noopDecompressor,
			expectError:     false,
			expectFromCache: false,
		},
		{
			name:        "range reader error",
			reader:      &mockRangeReader{err: errors.New("read failed")},
			header:      fakeHeader("fails-bipidibapidi"),
			ranger:      mockRanger{1337, 31337},
			decompress:  noopDecompressor,
			expectError: true,
		},
		{
			name: "decompression error",
			reader: &mockRangeReader{
				data: map[string][]byte{
					"1337:31337": generateFakeDirectoryData(10),
				},
			},
			header:      fakeHeader("fails-horrible"),
			ranger:      mockRanger{1337, 31337},
			decompress:  errorDecompressor,
			expectError: true,
		},
	}

	sfx := singleflight.NewShardedGroup[string, Directory](singleflight.WithShardCount(3))
	cache, err := NewOtterCache()
	if err != nil {
		t.Fatalf("instantiating cache")
	}
	ctx := t.Context()
	repo, err := NewDirectoryRepository(cache, sfx)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key := fmt.Sprintf("%s:%d:%d", tc.header.Etag, tc.ranger.Offset(), tc.ranger.Length())

			dir, _, err := repo.DirectoryAt(ctx, tc.header, tc.reader, tc.ranger, tc.decompress)

			if tc.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tc.expectError && !tc.expectFromCache {
				cached, ok := repo.cache.Get(ctx, key)
				if !ok || cached.Key() != dir.Key() {
					t.Errorf("expected directory to be cached under key %s", key)
				}
			}
		})
	}
}

func writeUvarint(buf *bytes.Buffer, val uint64) {
	// enough space for largest possible encoding of uin64
	var tmp [10]byte
	n := binary.PutUvarint(tmp[:], val)
	buf.Write(tmp[:n])
}

type mockRangeReader struct {
	data map[string][]byte
	err  error
}

func (m *mockRangeReader) ReadRange(_ context.Context, r Ranger) (io.ReadCloser, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := fmt.Sprintf("%d:%d", r.Offset(), r.Length())
	data := m.data[key]
	return io.NopCloser(bytes.NewReader(data)), nil
}

type mockRanger struct {
	offset uint64
	size   uint64
}

func (m mockRanger) Offset() uint64  { return m.offset }
func (m mockRanger) Length() uint64  { return m.size }
func (m mockRanger) Validate() error { return nil }

func fakeHeader(etag string) HeaderV3 {
	return HeaderV3{
		Etag:                etag,
		InternalCompression: CompressionNone,
	}
}

func noopDecompressor(r io.ReadCloser, _ Compression) (io.ReadCloser, error) {
	return io.NopCloser(r), nil
}

func errorDecompressor(r io.ReadCloser, _ Compression) (io.ReadCloser, error) {
	return nil, errors.New("failed to decompress")
}

func generateFakeDirectoryData(n int) []byte {
	var buf bytes.Buffer

	_ = binary.Write(&buf, binary.LittleEndian, uint64(n))

	writeUvarints := func(vals []uint64) {
		for _, v := range vals {
			_ = binary.Write(&buf, binary.LittleEndian, v)
		}
	}

	tileids := make([]uint64, n)
	runLens := make([]uint64, n)
	lengths := make([]uint64, n)
	offsets := make([]uint64, n)

	var lastID uint64
	var currentOffset uint64
	for i := range n {
		tileids[i] = uint64(rand.Intn(10) + 1)
		runLens[i] = uint64(rand.Intn(5) + 1)
		lengths[i] = uint64(rand.Intn(1024) + 1)
		offsets[i] = currentOffset
		currentOffset += lengths[i]
		lastID += tileids[i]
	}

	writeUvarints(tileids)
	writeUvarints(runLens)
	writeUvarints(lengths)
	writeUvarints(offsets)

	return buf.Bytes()
}

var (
	sinkEntry Entry
	sinkU64   uint64
)

func buildDirs(n uint64) *Directory {
	d := &Directory{size: n}
	d.entries = make(Entries, n)

	var id uint64
	for i := range n {
		id += uint64(rand.Intn(4) + 1)
		d.entries[i] = Entry{TileID: id, Offset: i, Length: i, RunLength: uint32(i%7) + 1}
	}
	return d
}

// precompute a set of random target keys spanning the whole id range
func buildTargets(d *Directory, count int) []uint64 {
	maxID := d.entries[len(d.entries)-1].TileID
	t := make([]uint64, count)
	for i := range t {
		t[i] = rand.Uint64() % (maxID + 1)
	}
	return t
}

func itoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func BenchmarkSumLengths(b *testing.B) {
	d := buildDirs(1 << 20)

	b.Run("object", func(b *testing.B) {
		for b.Loop() {
			var s uint64
			for i := range d.entries {
				s += d.entries[i].Length // pulls 32B line, uses 8B
			}
			sinkU64 = s
		}
	})
}

func BenchmarkFindEntry(b *testing.B) {
	sizes := []uint64{1 << 8, 1 << 10, 1 << 12, 1 << 14, 1 << 16, 1 << 18, 1 << 20}
	const targetCount = 4096
	const mask = targetCount - 1

	for _, n := range sizes {
		d := buildDirs(n)
		targets := buildTargets(d, targetCount)

		b.Run("struct-of-objects/N="+itoa(n), func(b *testing.B) {
			var i uint64
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				e := d.FindEntry(targets[i&mask])
				if e != nil {
					sinkEntry = *e
				}
				i++
			}
		})
	}
}
