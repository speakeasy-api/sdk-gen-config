package lockfile

import (
	"bufio"
	"bytes"
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
//   - Strip UTF-8 BOM only if present at the very beginning
//   - Convert CRLF and lone CR to LF
//   - Ignore the presence of a single trailing LF (drop it if present)
//
// When r is a *bytes.Reader, an optimized path avoids allocating a bufio.Reader
// and read buffer. For repeated calls with buffer reuse, use [NormalizedSHA1Hasher].
func HashNormalizedSHA1(r io.Reader) (string, error) {
	if br, ok := r.(*bytes.Reader); ok {
		data := make([]byte, br.Len())
		if _, err := io.ReadFull(br, data); err != nil {
			return "", err
		}
		return hashNormalizedSlice(data, nil, nil)
	}

	br := bufio.NewReaderSize(r, 64*1024)
	return hashNormalizedReader(br, nil, nil, nil)
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
