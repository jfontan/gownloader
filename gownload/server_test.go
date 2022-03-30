package gownload

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

type testReader struct {
	blockSize int
	blocks    int
	extra     int
	pos       int
}

func (r *testReader) blocksSize() int {
	return r.blockSize * r.blocks
}

func (r *testReader) totalSize() int {
	return r.blocksSize() + r.extra
}

func (r *testReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	i := 0
	for {
		if i >= len(p) {
			return i, nil
		}

		if r.pos >= r.totalSize() {
			return i, io.EOF
		}

		if r.pos >= r.blocksSize() {
			p[i] = 0
		} else {
			p[i] = byte(r.pos / r.blockSize)
		}

		i++
		r.pos++
	}
}

func (r *testReader) Seek(offset int64, whence int) (int64, error) {
	var pos int
	switch whence {
	case io.SeekStart:
		pos = int(offset)

	case io.SeekCurrent:
		pos = r.pos + int(offset)

	case io.SeekEnd:
		pos = r.totalSize() + int(offset)

	default:
		return int64(r.pos), os.ErrInvalid
	}

	if pos < 0 {
		return int64(r.pos), io.EOF
	}

	r.pos = pos
	return int64(pos), nil
}

func createTestServer(bs, b, e int) *http.Server {
	reader := &testReader{
		blockSize: bs,
		blocks:    b,
		extra:     e,
	}

	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "file.bin", time.Now(), reader)
	})

	s := &http.Server{
		Handler: m,
	}

	return s
}

func startTestServer(bs, b, e int) (string, func()) {
	d, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	s := createTestServer(bs, b, e)
	go s.Serve(d)

	addr := fmt.Sprintf("http://%s/", d.Addr())

	return addr, func() {
		s.Shutdown(context.Background())
	}
}

func TestHeaders(t *testing.T) {
	addr, shutdown := startTestServer(16, 8, 64)
	defer shutdown()

	r, err := http.Get(addr)
	require.NoError(t, err)
	defer r.Body.Close()

	spew.Dump(r.Header)

	b, err := io.ReadAll(r.Body)
	require.NoError(t, err)

	spew.Dump(b)
	println(b)
}
