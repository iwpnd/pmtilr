package pmtilr

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"
)

func writeUvarint(buf *bytes.Buffer, val uint64) {
	// enough space for largest possible encoding of uin64
	var tmp [10]byte
	n := binary.PutUvarint(tmp[:], val)
	buf.Write(tmp[:n])
}

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

			readCloser, err := mockReader.ReadRange(context.Background(), mockRanger{1337, 31337})
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
					"1337:31337": fakeDirectoryData(),
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
					"1337:31337": fakeDirectoryData(),
				},
			},
			header:      fakeHeader("fails-horrible"),
			ranger:      mockRanger{1337, 31337},
			decompress:  errorDecompressor,
			expectError: true,
		},
	}

	ctx := t.Context()
	repo, err := NewRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key := fmt.Sprintf("%s:%d:%d", tc.header.Etag, tc.ranger.Offset(), tc.ranger.Length())

			dir, err := repo.DirectoryAt(ctx, tc.header, tc.reader, tc.ranger, tc.decompress)

			if tc.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// ensure .Set is written
			// ristretto is eventually consistent, meaning sets
			// a) can get rejected, b) may take time passing the LFU
			repo.cache.Wait()

			if !tc.expectError && !tc.expectFromCache {
				cached, ok := repo.cache.Get(key)
				if !ok || cached.Key() != dir.Key() {
					t.Errorf("expected directory to be cached under key %s", key)
				}
			}
		})
	}
}

func BenchmarkDeserializeIsGzipReader(b *testing.B) {
	raw := generateFakeDirectoryData(10_000)
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(raw); err != nil {
		b.Fatalf("gzip write failed: %v", err)
	}
	if err := gw.Close(); err != nil {
		b.Fatalf("gzip close failed: %v", err)
	}
	compressed := buf.Bytes()
	r := bytes.NewReader(compressed)
	gr, err := gzip.NewReader(r)
	if err != nil {
		b.Fatalf("gzip NewReader failed: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		d := &Directory{}
		_ = d.deserialize(gr)
	}
}

func BenchmarkDeserializeIsByteReader(b *testing.B) {
	data := generateFakeDirectoryData(10_000)
	br := bytes.NewReader(data)

	b.ResetTimer()
	for b.Loop() {
		d := &Directory{}
		_ = d.deserialize(br)
	}
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

func noopDecompressor(r io.Reader, _ Compression) (io.ReadCloser, error) {
	return io.NopCloser(r), nil
}

func errorDecompressor(r io.Reader, _ Compression) (io.ReadCloser, error) {
	return nil, errors.New("failed to decompress")
}

func fakeDirectoryData() []byte {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, uint64(1))   // 1 entry
	_ = binary.Write(buf, binary.LittleEndian, uint64(1))   // delta
	_ = binary.Write(buf, binary.LittleEndian, uint64(2))   // runLength
	_ = binary.Write(buf, binary.LittleEndian, uint64(100)) // size
	_ = binary.Write(buf, binary.LittleEndian, uint64(500)) // offset
	return buf.Bytes()
}

func generateFakeDirectoryData(n int) []byte {
	var buf bytes.Buffer

	_ = binary.Write(&buf, binary.LittleEndian, uint64(n))

	writeUvarints := func(vals []uint64) {
		for _, v := range vals {
			_ = binary.Write(&buf, binary.LittleEndian, v)
		}
	}

	deltas := make([]uint64, n)
	runLens := make([]uint64, n)
	sizes := make([]uint64, n)
	offsets := make([]uint64, n)

	var lastID uint64
	var currentOffset uint64
	for i := range n {
		deltas[i] = uint64(rand.Intn(10) + 1)
		runLens[i] = uint64(rand.Intn(5) + 1)
		sizes[i] = uint64(rand.Intn(1024) + 1)
		offsets[i] = currentOffset
		currentOffset += sizes[i]
		lastID += deltas[i]
	}

	writeUvarints(deltas)
	writeUvarints(runLens)
	writeUvarints(sizes)
	writeUvarints(offsets)

	return buf.Bytes()
}
