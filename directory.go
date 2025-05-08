package pmtilr

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"sort"

	"github.com/dgraph-io/ristretto/v2"
)

const (
	cacheKeyTemplate = "%s:%d:%d" // etag:offset:size
)

type Entry struct {
	TileID    uint64 `json:"tile_id"`
	Offset    uint64 `json:"offset"`
	Size      uint64 `json:"size"`
	RunLength uint32 `json:"run_length"`
}

func (e Entry) String() string {
	jsonBytes, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return `{"error": "failed to marshal entry"}`
	}
	return string(jsonBytes)
}

func NewDirectory(
	ctx context.Context,
	header HeaderV3,
	reader RangeReader,
	ranger Ranger,
	decompress DecompressFunc,
) (*Directory, error) {
	data, err := reader.ReadRange(
		ctx,
		ranger,
	)
	if err != nil {
		return &Directory{}, fmt.Errorf("reading directory from source: %w", err)
	}

	decompReader, err := decompress(bytes.NewReader(data), header.InternalCompression)
	if err != nil {
		return &Directory{}, fmt.Errorf("decompressing directory: %w", err)
	}
	if closer, ok := decompReader.(io.Closer); ok {
		defer func() {
			if cerr := closer.Close(); cerr != nil {
				if err == nil {
					err = fmt.Errorf("closing decompressed reader: %w", cerr)
				}
			}
		}()
	}

	dir := &Directory{}
	if err := dir.deserialize(decompReader); err != nil {
		return &Directory{}, fmt.Errorf("deserializing directory: %w", err)
	}

	dir.key = fmt.Sprintf("%s:%d:%d", header.Etag, ranger.Offset(), ranger.Size())

	return dir, nil
}

type Directory struct {
	key     string
	size    uint64
	entries []Entry
}

func (d *Directory) Key() string {
	return d.key
}

func (d *Directory) Size() uint64 {
	return d.size
}

func (d *Directory) IterEntries() iter.Seq[Entry] {
	return func(yield func(Entry) bool) {
		for _, v := range d.entries {
			if !yield(v) {
				return
			}
		}
	}
}

func (d *Directory) FindTile(tileId uint64) (*Entry, error) {
	// Binary search for the first entry whose tileId > target.
	i := sort.Search(len(d.entries), func(i int) bool {
		return d.entries[i].TileID > tileId
	})

	// If i==0, every entries[j].tileId > tileId so no match.
	if i == 0 {
		return &Entry{}, fmt.Errorf("tileId %d not found", tileId)
	}

	// Candidate is the one just before that.
	e := d.entries[i-1]

	// Check exact match or run‑length cover:
	if tileId == e.TileID || tileId < e.TileID+uint64(e.RunLength) {
		return &e, nil
	}
	return &Entry{}, fmt.Errorf("tileId %d not found", tileId)
}

func (d *Directory) deserialize(r io.Reader) (err error) {
	br := bufio.NewReader(r)
	countEntries, err := binary.ReadUvarint(br)
	if err != nil {
		return fmt.Errorf("reading directory entries count: %w", err)
	}

	entries := make([]Entry, countEntries)
	d.entries = entries

	var lastId uint64
	for i := range d.entries {
		delta, err := binary.ReadUvarint(br)
		if err != nil {
			return fmt.Errorf("reading tileId delta at %d: %w", i, err)
		}
		d.entries[i].TileID = lastId + delta
		lastId = d.entries[i].TileID
	}

	for i := range d.entries {
		runLength, err := binary.ReadUvarint(br)
		if err != nil {
			return fmt.Errorf("reading runLength at %d: %w", i, err)
		}
		d.entries[i].RunLength = uint32(runLength) //nolint:gosec
	}

	for i := range d.entries {
		size, err := binary.ReadUvarint(br)
		if err != nil {
			return fmt.Errorf("reading length at %d: %w", i, err)
		}
		d.entries[i].Size = size
	}

	for i := range d.entries {
		offset, err := binary.ReadUvarint(br)
		if err != nil {
			return fmt.Errorf("reading offset at %d: %w", i, err)
		}
		if offset == 0 && i > 0 {
			d.entries[i].Offset = d.entries[i-1].Offset + d.entries[i-1].Size
		} else {
			d.entries[i].Offset = offset - 1
		}
	}

	d.size = countEntries

	return
}

// NOTE: will have options eventually
func NewRepository() (*Repository, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, *Directory]{
		NumCounters: 10 * 500 * 1024, // number of keys to track frequency of (10M).
		MaxCost:     500 * 1024,      // 500mb
		BufferItems: 64,              // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	dirs := &Repository{
		cache: cache,
	}

	return dirs, nil
}

type Repository struct {
	cache *ristretto.Cache[string, *Directory]
}

func (d *Repository) DirectoryAt(
	ctx context.Context,
	header HeaderV3,
	reader RangeReader,
	ranger Ranger,
	decompress DecompressFunc,
) (*Directory, error) {
	key := fmt.Sprintf(cacheKeyTemplate, header.Etag, ranger.Offset(), ranger.Size())
	dir, ok := d.cache.Get(key)
	if ok {
		return dir, nil
	}
	dir, err := NewDirectory(ctx, header, reader, ranger, decompress)
	if err != nil {
		return &Directory{}, err
	}

	// NOTE: even if it fails once, eventually it succeeds
	// ristretto is eventually consistent
	_ = d.cache.Set(key, dir, 1)

	return dir, nil
}

func (d *Repository) Tile(
	ctx context.Context,
	header HeaderV3,
	reader RangeReader,
	decompress DecompressFunc, z, x, y uint64,
) ([]byte, error) {
	if z < uint64(header.MinZoom) && z > uint64(header.MaxZoom) {
		return []byte{}, fmt.Errorf(
			"invalid zoom: %d for allowed range of %d to %d",
			z,
			header.MinZoom,
			header.MaxZoom,
		)
	}
	tileId, err := ZxyToHilbertTileID(z, x, y)
	if err != nil {
		return []byte{}, fmt.Errorf("resolving hilbert tile id from z: %d x: %d y: %d", z, x, y)
	}

	dO := header.RootOffset
	dS := header.RootLength
	for range 3 {
		dir, err := d.DirectoryAt(ctx, header, reader, NewRange(dO, dS), decompress)
		if err != nil {
			return []byte{}, err
		}
		entry, err := dir.FindTile(tileId)
		if err != nil {
			return []byte{}, err
		}
		// TODO: refactor
		if entry != nil {
			if entry.RunLength > 0 {
				data, err := reader.ReadRange(
					ctx,
					NewRange(header.TileDataOffset+entry.Offset, entry.Size),
				)
				if err != nil {
					return []byte{}, err
				}
				decompReader, err := decompress(bytes.NewReader(data), header.TileCompression)
				if err != nil {
					return []byte{}, fmt.Errorf("decompressing tile entry: %w", err)
				}

				tileData, err := io.ReadAll(decompReader)
				if err != nil {
					return []byte{}, fmt.Errorf("reading decompressed metadata: %w", err)
				}

				if closer, ok := decompReader.(io.Closer); ok {
					cerr := closer.Close()
					if cerr != nil {
						return []byte{}, fmt.Errorf("closing decompression reader: %w", cerr)
					}
				}

				return tileData, nil
			}
			dO = header.LeafDirectoryOffset + entry.Offset
			dS = entry.Size
		} else {
			return []byte{}, nil
		}
	}
	return []byte{}, fmt.Errorf("maximum directory depth exceeded, ")
}

func (d *Repository) Flush() {
	d.cache.Clear()
}

func (d *Repository) Close() {
	d.cache.Close()
}
