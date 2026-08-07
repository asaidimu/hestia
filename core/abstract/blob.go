package abstract

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var ErrChecksumMismatch = errors.New("blob: chunk checksum mismatch")

// Blob represents a payload chunk or stream moving through the pipeline.
// Memory layout is optimized for 64-bit alignment (0 padding bytes, 120B total).
type Blob struct {
	Data []byte

	ContentType   string
	Checksum      string
	TotalChecksum string
	Stream        io.Reader

	Offset    int64
	Size      int64
	TotalSize int64
	Release   func()
}

// Reader returns an io.Reader over the chunk payload.
func (b *Blob) Reader() io.Reader {
	if b.Stream != nil {
		return b.Stream
	}
	if len(b.Data) > 0 {
		return bytes.NewReader(b.Data)
	}
	return bytes.NewReader(nil)
}

// ChunkSize returns the actual byte length of this chunk.
func (b *Blob) ChunkSize() int64 {
	if b.Size > 0 {
		return b.Size
	}
	return int64(len(b.Data))
}

// Verify validates b.Data against b.Checksum if a checksum is present.
func (b *Blob) Verify() error {
	if b.Checksum == "" || len(b.Data) == 0 {
		return nil
	}
	hash := sha256.Sum256(b.Data)
	sum := hex.EncodeToString(hash[:])
	if !equalFold(sum, b.Checksum) {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, b.Checksum, sum)
	}
	return nil
}

// Free executes the Release hook if set, then clears references to free memory.
func (b *Blob) Free() {
	if b.Release != nil {
		b.Release()
		b.Release = nil
	}
	b.Data = nil
	b.Stream = nil
}

func equalFold(s1, s2 string) bool {
	return len(s1) == len(s2) && (s1 == s2 || bytes.EqualFold([]byte(s1), []byte(s2)))
}
