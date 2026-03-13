package lockfile

import (
	"bufio"
	"bytes"
	"crypto/sha1" // nolint:gosec // sha1 is intentional as we're effectively copying git
	"encoding/hex"
	"hash"
	"io"
	"sync"
)

// hashState holds the reusable buffers for a single hash operation.
type hashState struct {
	h   hash.Hash
	out []byte
	// Only used for io.Reader (non-bytes.Reader) inputs:
	br *bufio.Reader
	in []byte
}

// NormalizedSHA1Hasher computes normalized SHA1 checksums with buffer reuse.
// It is safe for concurrent use.
type NormalizedSHA1Hasher struct {
	pool sync.Pool
}

// NewNormalizedSHA1Hasher returns a new hasher that pools internal buffers
// across calls. A single hasher can be shared across goroutines.
func NewNormalizedSHA1Hasher() *NormalizedSHA1Hasher {
	nh := &NormalizedSHA1Hasher{}
	nh.pool.New = func() any {
		return &hashState{
			h:   sha1.New(),
			out: make([]byte, 0, 64*1024),
		}
	}
	return nh
}

// HashNormalizedSHA1 computes SHA1 over a canonicalized stream with the same
// normalization rules as the standalone [HashNormalizedSHA1] function. Buffers
// are reused across calls via an internal pool.
func (nh *NormalizedSHA1Hasher) HashNormalizedSHA1(r io.Reader) (string, error) {
	s := nh.pool.Get().(*hashState)
	defer nh.pool.Put(s)
	s.h.Reset()
	s.out = s.out[:0]

	if br, ok := r.(*bytes.Reader); ok {
		data := make([]byte, br.Len())
		if _, err := io.ReadFull(br, data); err != nil {
			return "", err
		}
		return hashNormalizedSlice(data, s.h, &s.out)
	}

	if s.br == nil {
		s.br = bufio.NewReaderSize(r, 64*1024)
		s.in = make([]byte, 64*1024)
	} else {
		s.br.Reset(r)
	}
	return hashNormalizedReader(s.br, s.h, s.in, &s.out)
}

// normalizer implements the shared line-ending normalization state machine.
// It converts CRLF and lone CR to LF, and drops a single trailing LF.
// Normalized bytes are buffered in out and flushed to h periodically.
type normalizer struct {
	h           hash.Hash
	out         *[]byte
	prevCR      bool
	pending     byte
	havePending bool
}

func (n *normalizer) flush() error {
	if len(*n.out) == 0 {
		return nil
	}
	if _, err := n.h.Write(*n.out); err != nil {
		return err
	}
	*n.out = (*n.out)[:0]
	return nil
}

func (n *normalizer) emit(b byte) error {
	if n.havePending {
		*n.out = append(*n.out, n.pending)
		if len(*n.out) >= 32*1024 {
			if err := n.flush(); err != nil {
				return err
			}
		}
	}
	n.pending = b
	n.havePending = true
	return nil
}

// writeByte processes a single byte through the normalization state machine.
func (n *normalizer) writeByte(c byte) error {
	if n.prevCR {
		if err := n.emit('\n'); err != nil {
			return err
		}
		n.prevCR = false
		if c == '\n' {
			return nil // CRLF -> single LF already emitted
		}
	}

	if c == '\r' {
		n.prevCR = true
		return nil
	}
	return n.emit(c)
}

// finish flushes any remaining state. Must be called after all bytes have been
// written. Returns the hex-encoded hash.
func (n *normalizer) finish() (string, error) {
	if n.prevCR {
		if err := n.emit('\n'); err != nil {
			return "", err
		}
	}

	// Drop exactly one trailing LF.
	if n.havePending && n.pending != '\n' {
		*n.out = append(*n.out, n.pending)
	}
	if err := n.flush(); err != nil {
		return "", err
	}

	return hex.EncodeToString(n.h.Sum(nil)), nil
}

// hashNormalizedSlice normalizes data from a byte slice and writes the result
// to h. If h is nil, a new SHA1 hash is allocated. out is used as scratch
// space for buffering writes to the hash; the pointer is updated if the slice
// grows. Returns the hex-encoded hash.
func hashNormalizedSlice(data []byte, h hash.Hash, out *[]byte) (string, error) {
	if h == nil {
		h = sha1.New()
	}
	if out == nil {
		buf := make([]byte, 0, 64*1024)
		out = &buf
	}

	// Strip UTF-8 BOM if present at the start.
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	n := normalizer{h: h, out: out}
	for _, c := range data {
		if err := n.writeByte(c); err != nil {
			return "", err
		}
	}
	return n.finish()
}

// hashNormalizedReader normalizes data from a buffered reader and writes the
// result to h. If h is nil, a new SHA1 hash is allocated. in is the read
// buffer; if nil, a new one is allocated. out is used as scratch space; the
// pointer is updated if the slice grows. Returns the hex-encoded hash.
func hashNormalizedReader(br *bufio.Reader, h hash.Hash, in []byte, out *[]byte) (string, error) {
	if h == nil {
		h = sha1.New()
	}
	if in == nil {
		in = make([]byte, 64*1024)
	}
	if out == nil {
		buf := make([]byte, 0, 64*1024)
		out = &buf
	}

	// Strip UTF-8 BOM if present at the very start.
	if b, err := br.Peek(3); err == nil && len(b) >= 3 &&
		b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		_, _ = br.Discard(3)
	}

	n := normalizer{h: h, out: out}
	for {
		nr, err := br.Read(in)
		if nr > 0 {
			for _, c := range in[:nr] {
				if writeErr := n.writeByte(c); writeErr != nil {
					return "", writeErr
				}
			}
			if flushErr := n.flush(); flushErr != nil {
				return "", flushErr
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
	}

	return n.finish()
}
