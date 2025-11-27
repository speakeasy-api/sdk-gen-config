package lockfile

import (
	"bufio"
	"crypto/sha1" // nolint:gosec // sha1 is intentional as we're effectively copying git
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
)

// ComputeFileChecksum returns a checksum string like "sha1:<hex>"
// by hashing the normalized contents of root/relPath using the provided filesystem.
func ComputeFileChecksum(fileSystem fs.FS, relPath string) (string, error) {
	f, err := fileSystem.Open(relPath)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", relPath, err)
	}
	defer f.Close()

	sumHex, err := HashNormalizedSHA1(f)
	if err != nil {
		return "", fmt.Errorf("hash %s: %w", relPath, err)
	}
	return "sha1:" + sumHex, nil
}

// HashNormalizedSHA1 computes SHA1 over a canonicalized stream:
// - Strip UTF-8 BOM only if present at the very beginning
// - Convert CRLF and lone CR to LF
// - Ignore the presence of a single trailing LF (drop it if present)
func HashNormalizedSHA1(r io.Reader) (string, error) {
	br := bufio.NewReaderSize(r, 64*1024)

	// Strip UTF-8 BOM if present at the very start.
	if b, err := br.Peek(3); err == nil && len(b) >= 3 &&
		b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		_, _ = br.Discard(3)
	}

	h := sha1.New()

	// State across chunks
	const readBufSize = 64 * 1024
	in := make([]byte, readBufSize)
	out := make([]byte, 0, readBufSize)

	var prevCR bool      // previous byte was '\r' not yet emitted
	var pending byte     // last normalized byte not yet written (for final-LF handling)
	var havePending bool // whether pending is valid

	flushOut := func() error {
		if len(out) == 0 {
			return nil
		}
		if _, err := h.Write(out); err != nil {
			return err
		}
		out = out[:0]
		return nil
	}

	emit := func(b byte) error {
		// Buffer everything except keep the last byte in pending.
		if havePending {
			out = append(out, pending)
			if len(out) >= 32*1024 {
				if err := flushOut(); err != nil {
					return err
				}
			}
		}
		pending = b
		havePending = true
		return nil
	}

	for {
		n, err := br.Read(in)
		if n > 0 {
			buf := in[:n]
			for i := 0; i < len(buf); i++ {
				c := buf[i]

				if prevCR {
					if c == '\n' {
						// CRLF -> emit LF once
						if err := emit('\n'); err != nil {
							return "", err
						}
						prevCR = false
						continue
					}
					// Lone CR -> treat as newline
					if err := emit('\n'); err != nil {
						return "", err
					}
					prevCR = false
					// fallthrough to handle current c normally
				}

				if c == '\r' {
					prevCR = true
					continue
				}
				// Normal path: pass through, but normalize LF as-is
				if err := emit(c); err != nil {
					return "", err
				}
			}
			// Flush buffered out bytes opportunistically
			if err := flushOut(); err != nil {
				return "", err
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
	}

	// Handle final pending states.
	if prevCR {
		// File ended with CR -> normalize to LF
		if err := emit('\n'); err != nil {
			return "", err
		}
	}

	// Flush everything but drop exactly one final LF if present.
	if havePending && pending != '\n' {
		out = append(out, pending)
	}
	if err := flushOut(); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// PopulateMissingChecksums computes last_write_checksum for any TrackedFiles entries
// where LastWriteChecksum is empty. The fileSystem should be rooted at the directory containing
// the generated files (parent of .speakeasy/).
func PopulateMissingChecksums(lf *LockFile, fileSystem fs.FS) error {
	if lf.TrackedFiles == nil {
		return nil
	}

	for path := range lf.TrackedFiles.Keys() {
		tf, ok := lf.TrackedFiles.Get(path)
		if !ok {
			continue
		}

		if tf.LastWriteChecksum == "" {
			checksum, err := ComputeFileChecksum(fileSystem, path)
			if err != nil {
				continue
			}
			tf.LastWriteChecksum = checksum
			lf.TrackedFiles.Set(path, tf)
		}
	}
	return nil
}
