package pmtilr

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"sort"
	"sync"

	"github.com/dgraph-io/ristretto/v2"
)

const directoryMaxDepth = 3

var readerPool = sync.Pool{
	New: func() any {
		// allocate a *bytes.Reader with zero‐length backing slice
		return new(bufio.Reader)
	},
}

func acquireReader(newReader io.Reader) *bufio.Reader {
	r := readerPool.Get().(*bufio.Reader) //nolint:errcheck,forcetypeassert
	r.Reset(newReader)
	return r
}

func releaseReader(usedReader *bufio.Reader) {
	readerPool.Put(usedReader)
}

// Entry holds a reference to the exact location of a tile within
// the PMTiles archive.
// Each entry describes either where a specific tile can be found in the tile data
// section or where a leaf directory can be found in the leaf directories section.
type Entry struct {
	TileID    uint64 `json:"tile_id"`
	Offset    uint64 `json:"offset"`
	Length    uint64 `json:"length"`
	RunLength uint32 `json:"run_length"`
}

// Entries represents a slice of Entry values.
// It provides methods to deserialize entry attributes in order.
type Entries []Entry

// readEntries reads a list of Entry records from the provided buffered reader.
//
// It expects the data to be Uvarint-encoded in the following order:
// 1. Entry count
// 2. TileID deltas (relative to previous ID)
// 3. Run lengths
// 4. Lengths (in bytes)
// 5. Offsets (0 means propagate from previous offset + previous length)
//
// The deserialization logic respects the PMTiles binary layout and handles
// offset propagation.
//
// Returns a fully populated Entries slice, or an error if decoding fails.
func readEntries(br *bufio.Reader) (entries Entries, err error) {
	countEntries, err := binary.ReadUvarint(br)
	if err != nil {
		return nil, fmt.Errorf("reading directory entries count: %w", err)
	}

	entries = make([]Entry, countEntries)
	err = entries.deserialize(br)
	if err != nil {
		return entries, err
	}

	return
}

// String returns a JSON string of the Entry.
func (e Entry) String() string {
	jsonBytes, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return `{"error": "failed to marshal entry"}`
	}
	return string(jsonBytes)
}

// deserialize populates the Entries slice by reading tile ID deltas,
// runlengths, lengths, and offsets from the given reader.
func (e Entries) deserialize(br *bufio.Reader) (err error) {
	if e == nil {
		return fmt.Errorf("cannot deserialize a nil array")
	}

	deserializeInOrder := []func(*bufio.Reader) error{
		e.addTileID,
		e.addRunLength,
		e.addLength,
		e.addOffset,
	}

	for _, fn := range deserializeInOrder {
		err = fn(br)
		if err != nil {
			return err
		}
	}

	return
}

// addTileID reads and decodes tile ID deltas from the reader.
// Each value is added to the previous tile ID to compute the full sequence.
//
// Example:
//
//	delta1 = 3 → TileID[0] = 3
//	delta2 = 1 → TileID[1] = 4 (3 + 1)
func (e Entries) addTileID(br *bufio.Reader) (err error) {
	var lastId uint64
	for i := range e {
		delta, err := binary.ReadUvarint(br)
		if err != nil {
			return fmt.Errorf("reading tileId delta at %d: %w", i, err)
		}
		e[i].TileID = lastId + delta
		lastId = e[i].TileID
	}
	return
}

// addRunLength reads and assigns run lengths for each entry.
// Each run length is Uvarint-encoded and cast to uint32.
func (e Entries) addRunLength(br *bufio.Reader) (err error) {
	for i := range e {
		runLength, err := binary.ReadUvarint(br)
		if err != nil {
			return fmt.Errorf("reading runLength at %d: %w", i, err)
		}
		e[i].RunLength = uint32(runLength) //nolint:gosec
	}
	return
}

// addLength reads and assigns the byte length for each tile entry.
// Length values are Uvarint-encoded.
func (e Entries) addLength(br *bufio.Reader) (err error) {
	for i := range e {
		length, err := binary.ReadUvarint(br)
		if err != nil {
			return fmt.Errorf("reading length at %d: %w", i, err)
		}
		e[i].Length = length
	}
	return
}

// addOffset reads and assigns byte offsets for each entry.
//
// Offsets are Uvarint-encoded. A value of 0 (except for the first entry)
// triggers offset propagation: the current offset is set to the previous
// entry’s offset plus its length.
//
// The PMTiles format stores offsets as (offset + 1), so actual offset = stored - 1.
func (e Entries) addOffset(br *bufio.Reader) (err error) {
	for i := range e {
		offset, err := binary.ReadUvarint(br)
		if err != nil {
			return fmt.Errorf("reading offset at %d: %w", i, err)
		}
		if offset == 0 && i > 0 {
			// previous offset + previous length
			e[i].Offset = e[i-1].Offset + e[i-1].Length
		} else {
			e[i].Offset = offset - 1
		}
	}
	return
}

// NewDirectory creates a new Directory. A directory is a collection of
// entries that can be resolved from the `header.RootDirectoryOffset` of the PMTiles
// when the requested directory is a root directory. Otherwise the directory
// is fetched from the `header.LeafDirectoryOffset`
func NewDirectory(
	ctx context.Context,
	header HeaderV3,
	reader RangeReader,
	ranger Ranger,
	decompress DecompressFunc,
) (*Directory, error) {
	rangeReader, err := reader.ReadRange(
		ctx,
		ranger,
	)
	if err != nil {
		return &Directory{}, fmt.Errorf("reading directory from source: %w", err)
	}
	defer func() {
		if cerr := rangeReader.Close(); cerr != nil {
			if err == nil {
				err = cerr
			} else {
				err = errors.Join(err, cerr)
			}
		}
	}()

	decompReader, err := decompress(rangeReader, header.InternalCompression)
	if err != nil {
		return &Directory{}, fmt.Errorf("decompressing directory: %w", err)
	}

	defer func() {
		if cerr := decompReader.Close(); cerr != nil {
			if err == nil {
				err = fmt.Errorf("closing decompressed reader: %w", cerr)
			} else {
				err = errors.Join(err, fmt.Errorf("closing decompressed reader: %w", cerr))
			}
		}
	}()

	dir := &Directory{}
	if err := dir.deserialize(decompReader); err != nil {
		return &Directory{}, fmt.Errorf("deserializing directory: %w", err)
	}

	dir.key = fmt.Sprintf("%s:%d:%d", header.Etag, ranger.Offset(), ranger.Length())

	return dir, nil
}

// Directory is a collection of Tile Entries.
type Directory struct {
	key     string
	size    uint64
	entries Entries
}

// Key returns the Directory key.
func (d *Directory) Key() string {
	return d.key
}

// Size returns the Directory size.
func (d *Directory) Size() uint64 {
	return d.size
}

// IterEntries is an iterator over the entries of a directory.
func (d *Directory) IterEntries() iter.Seq[Entry] {
	return func(yield func(Entry) bool) {
		for _, v := range d.entries {
			if !yield(v) {
				return
			}
		}
	}
}

// FindTile resolves an Entry by tileID.
func (d *Directory) FindTile(tileId uint64) *Entry {
	// Binary search for the first entry whose tileId > target.
	i := sort.Search(len(d.entries), func(i int) bool {
		return d.entries[i].TileID > tileId
	})

	// every entries[j].tileId > tileId so no match.
	if i == 0 {
		return nil
	}

	// all entries at or after i have TileIDs greater than tileId
	// therefor candidate is the one just before that.
	e := &d.entries[i-1]

	// entry is a directory and should be traversed further
	if e.RunLength == 0 {
		return e
	}

	// Check exact match or run‑length cover:
	if tileId == e.TileID || tileId < e.TileID+uint64(e.RunLength) {
		return e
	}

	// not found
	return nil
}

// deserialize the directory from a decompression reader entry by entry.
func (d *Directory) deserialize(r io.Reader) (err error) {
	br := acquireReader(r)
	defer releaseReader(br)

	entries, err := readEntries(br)
	if err != nil {
		return err
	}

	d.entries = entries
	d.size = uint64(len(entries))

	return
}

// NOTE: will have options eventually
func NewRepository() (*Repository, error) {
	cache, err := NewDefaultCache()
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
	key := fmt.Sprintf(cacheKeyTemplate, header.Etag, ranger.Offset(), ranger.Length())
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
	tileId, err := FastZXYToHilbertTileID(z, x, y)
	if err != nil {
		return []byte{}, fmt.Errorf("resolving hilbert tile id from z: %d x: %d y: %d", z, x, y)
	}

	dO := header.RootOffset
	dS := header.RootLength
	for range directoryMaxDepth {
		dir, err := d.DirectoryAt(ctx, header, reader, NewRange(dO, dS), decompress)
		if err != nil {
			return []byte{}, err
		}
		entry := dir.FindTile(tileId)
		if entry != nil {
			// entry is not a directory and we can attempt to read the tile data
			if entry.RunLength > 0 {
				rangeReader, err := reader.ReadRange(
					ctx,
					NewRange(header.TileDataOffset+entry.Offset, entry.Length),
				)
				if err != nil {
					return []byte{}, err
				}

				decompReader, err := decompress(rangeReader, header.TileCompression)
				if err != nil {
					rangeReader.Close() //nolint:errcheck
					return []byte{}, fmt.Errorf("decompressing tile entry: %w", err)
				}

				tileData, err := io.ReadAll(decompReader)
				rangeReader.Close() //nolint:errcheck
				if err != nil {
					return []byte{}, fmt.Errorf("reading decompressed metadata: %w", err)
				}

				if cerr := decompReader.Close(); cerr != nil {
					return []byte{}, fmt.Errorf("closing decompression reader: %w", cerr)
				}

				return tileData, nil
			}
			// entry is a directory and we want to traverse the directory tree further
			dO = header.LeafDirectoryOffset + entry.Offset
			dS = entry.Length
		} else {
			return []byte{}, nil
		}
	}
	return []byte{}, fmt.Errorf("maximum directory depth exceeded")
}

func (d *Repository) Flush() {
	d.cache.Clear()
}

func (d *Repository) Close() {
	d.cache.Close()
}
