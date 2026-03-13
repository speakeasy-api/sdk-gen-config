package lockfile_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/speakeasy-api/sdk-gen-config/lockfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goldenHash returns the hash from the standalone function for comparison.
func goldenHash(t *testing.T, data []byte) string {
	t.Helper()
	h, err := lockfile.HashNormalizedSHA1(bytes.NewReader(data))
	require.NoError(t, err)
	return h
}

func TestNormalizedSHA1Hasher_MatchesStandalone(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{"empty", []byte{}},
		{"simple text", []byte("hello world")},
		{"trailing LF", []byte("hello\n")},
		{"no trailing LF", []byte("hello")},
		{"CRLF", []byte("hello\r\nworld")},
		{"CRLF trailing", []byte("hello\r\nworld\r\n")},
		{"lone CR", []byte("hello\rworld")},
		{"lone CR trailing", []byte("hello\rworld\r")},
		{"mixed line endings", []byte("a\r\nb\rc\nd")},
		{"BOM prefix", append([]byte{0xEF, 0xBB, 0xBF}, []byte("hello")...)},
		{"BOM only", []byte{0xEF, 0xBB, 0xBF}},
		{"BOM with CRLF", append([]byte{0xEF, 0xBB, 0xBF}, []byte("a\r\nb\r\n")...)},
		{"multiple trailing LFs", []byte("hello\n\n")},
		{"only LF", []byte("\n")},
		{"only CRLF", []byte("\r\n")},
		{"only CR", []byte("\r")},
		{"binary-ish data", []byte{0x00, 0x01, 0x0D, 0x0A, 0xFF}},
	}

	hasher := lockfile.NewNormalizedSHA1Hasher()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := goldenHash(t, tt.input)

			got, err := hasher.HashNormalizedSHA1(bytes.NewReader(tt.input))
			require.NoError(t, err)
			assert.Equal(t, want, got, "hasher result should match standalone")
		})
	}
}

func TestNormalizedSHA1Hasher_ReuseSafety(t *testing.T) {
	hasher := lockfile.NewNormalizedSHA1Hasher()

	inputs := [][]byte{
		[]byte("first file content\r\n"),
		append([]byte{0xEF, 0xBB, 0xBF}, []byte("second with BOM\n")...),
		[]byte("third no newline"),
		{},
		bytes.Repeat([]byte("large content\r\n"), 10000),
	}

	for i, data := range inputs {
		want := goldenHash(t, data)
		got, err := hasher.HashNormalizedSHA1(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, want, got, "mismatch on call %d", i)
	}
}

func TestNormalizedSHA1Hasher_IOReader(t *testing.T) {
	hasher := lockfile.NewNormalizedSHA1Hasher()
	data := []byte("hello\r\nworld\n")

	// Use strings.Reader (not *bytes.Reader) to exercise the io.Reader path.
	got, err := hasher.HashNormalizedSHA1(strings.NewReader(string(data)))
	require.NoError(t, err)

	want := goldenHash(t, data)
	assert.Equal(t, want, got)
}

func TestNormalizedSHA1Hasher_IOReaderReuse(t *testing.T) {
	hasher := lockfile.NewNormalizedSHA1Hasher()

	inputs := []string{
		"first\r\n",
		"second\n",
		"third",
	}

	for _, s := range inputs {
		want := goldenHash(t, []byte(s))
		got, err := hasher.HashNormalizedSHA1(strings.NewReader(s))
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
}

func TestNormalizedSHA1Hasher_ConcurrentSafety(t *testing.T) {
	hasher := lockfile.NewNormalizedSHA1Hasher()
	const goroutines = 20
	const iterations = 50

	type result struct {
		hash string
		err  error
	}

	results := make(chan result, goroutines*iterations)

	data := []byte("concurrent test data\r\nwith CRLF\r\n")
	want := goldenHash(t, data)

	for range goroutines {
		go func() {
			for range iterations {
				h, err := hasher.HashNormalizedSHA1(bytes.NewReader(data))
				results <- result{h, err}
			}
		}()
	}

	for range goroutines * iterations {
		r := <-results
		require.NoError(t, r.err)
		assert.Equal(t, want, r.hash)
	}
}

func TestHashNormalizedSHA1_BytesReaderFastPath(t *testing.T) {
	// Verify the standalone function produces the same result whether given
	// a *bytes.Reader (fast path) or an io.Reader (buffered path).
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte("hello\r\nworld\r")...)

	fromBytesReader, err := lockfile.HashNormalizedSHA1(bytes.NewReader(data))
	require.NoError(t, err)

	fromIOReader, err := lockfile.HashNormalizedSHA1(strings.NewReader(string(data)))
	require.NoError(t, err)

	assert.Equal(t, fromBytesReader, fromIOReader)
}

func TestNormalizedSHA1Hasher_LargeData(t *testing.T) {
	// Generate data larger than the 64KB buffer to test chunked processing.
	var buf bytes.Buffer
	for i := range 5000 {
		switch i % 3 {
		case 0:
			buf.WriteString("line with CRLF ending\r\n")
		case 1:
			buf.WriteString("line with LF ending\n")
		default:
			buf.WriteString("line with CR ending\r")
		}
	}
	data := buf.Bytes()

	hasher := lockfile.NewNormalizedSHA1Hasher()
	want := goldenHash(t, data)

	got, err := hasher.HashNormalizedSHA1(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, want, got)

	// Also test via io.Reader path.
	got2, err := hasher.HashNormalizedSHA1(strings.NewReader(string(data)))
	require.NoError(t, err)
	assert.Equal(t, want, got2)
}

func FuzzNormalizedSHA1Hasher(f *testing.F) {
	f.Add([]byte("hello\r\nworld\n"))
	f.Add([]byte{0xEF, 0xBB, 0xBF, 'a', '\r', '\n'})
	f.Add([]byte{})
	f.Add([]byte("\r"))
	f.Add([]byte("\n"))
	f.Add([]byte("\r\n"))

	hasher := lockfile.NewNormalizedSHA1Hasher()

	f.Fuzz(func(t *testing.T, data []byte) {
		// Standalone via bytes.Reader (fast path)
		standalone, err := lockfile.HashNormalizedSHA1(bytes.NewReader(data))
		require.NoError(t, err)

		// Standalone via io.Reader (buffered path)
		standaloneIO, err := lockfile.HashNormalizedSHA1(strings.NewReader(string(data)))
		require.NoError(t, err)
		assert.Equal(t, standalone, standaloneIO, "standalone fast-path vs buffered-path mismatch")

		// Hasher via bytes.Reader
		hasherResult, err := hasher.HashNormalizedSHA1(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, standalone, hasherResult, "hasher vs standalone mismatch")

		// Hasher via io.Reader
		hasherIO, err := hasher.HashNormalizedSHA1(io.NopCloser(strings.NewReader(string(data))))
		require.NoError(t, err)
		assert.Equal(t, standalone, hasherIO, "hasher io.Reader vs standalone mismatch")
	})
}

func BenchmarkHashNormalizedSHA1_Standalone(b *testing.B) {
	data := bytes.Repeat([]byte("some generated code line\r\n"), 4000) // ~100KB
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = lockfile.HashNormalizedSHA1(bytes.NewReader(data))
	}
}

func BenchmarkHashNormalizedSHA1_Hasher(b *testing.B) {
	data := bytes.Repeat([]byte("some generated code line\r\n"), 4000) // ~100KB
	hasher := lockfile.NewNormalizedSHA1Hasher()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = hasher.HashNormalizedSHA1(bytes.NewReader(data))
	}
}

func BenchmarkHashNormalizedSHA1_Standalone_IOReader(b *testing.B) {
	data := string(bytes.Repeat([]byte("some generated code line\r\n"), 4000))
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = lockfile.HashNormalizedSHA1(strings.NewReader(data))
	}
}

func BenchmarkHashNormalizedSHA1_Hasher_IOReader(b *testing.B) {
	data := string(bytes.Repeat([]byte("some generated code line\r\n"), 4000))
	hasher := lockfile.NewNormalizedSHA1Hasher()
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = hasher.HashNormalizedSHA1(strings.NewReader(data))
	}
}
