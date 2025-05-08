package pmtilr

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"testing"
)

type mockRangeReader struct {
	data map[string][]byte
	err  error
}

func (m *mockRangeReader) ReadRange(_ context.Context, r Ranger) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := fmt.Sprintf("%d:%d", r.Offset(), r.Size())
	return m.data[key], nil
}

type mockRanger struct {
	offset uint64
	size   uint64
}

func (m mockRanger) Offset() uint64  { return m.offset }
func (m mockRanger) Size() uint64    { return m.size }
func (m mockRanger) Validate() error { return nil }

func fakeHeader(etag string) HeaderV3 {
	return HeaderV3{
		Etag:                etag,
		InternalCompression: CompressionNone,
	}
}

func noopDecompressor(r io.Reader, _ Compression) (io.Reader, error) {
	return r, nil
}

func errorDecompressor(r io.Reader, _ Compression) (io.Reader, error) {
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

func TestRepositoryDirectoryAt(t *testing.T) {
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

	ctx := context.Background()
	repo, err := NewRepository()
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key := fmt.Sprintf("%s:%d:%d", tc.header.Etag, tc.ranger.Offset(), tc.ranger.Size())

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

// --benmarks

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

func BenchmarkDeserializeOriginal(b *testing.B) {
	data := generateFakeDirectoryData(10_000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := &Directory{}
		_ = d.deserialize(bytes.NewReader(data))
	}
}
