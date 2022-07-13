package fasthttptest

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"io/ioutil"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

var zeroTCPAddr = &net.TCPAddr{
	IP: net.IPv4zero,
}

type testLogger struct {
	lock sync.Mutex
	out  string
}

func (cl *testLogger) Printf(format string, args ...interface{}) {
	cl.lock.Lock()
	cl.out += fmt.Sprintf(format, args...)[6:] + "\n"
	cl.lock.Unlock()
}

func TestServerCRNLAfterPost(t *testing.T) {
	t.Parallel()

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
		},
		Logger:      &testLogger{},
		ReadTimeout: time.Millisecond * 100,
	}

	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	go func() {
		if err := s.Serve(ln); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	c, err := ln.Dial()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer c.Close()
	if _, err = c.Write([]byte("POST / HTTP/1.1\r\nHost: golang.org\r\nContent-Length: 3\r\n\r\nABC" +
		"\r\n\r\n", // <-- this stuff is bogus, but we'll ignore it
	)); err != nil {
		t.Fatal(err)
	}

	br := bufio.NewReader(c)
	var resp fasthttp.Response
	if err := resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusOK)
	}
	if err := resp.Read(br); err == nil {
		t.Fatal("expected error") // We didn't send a request so we should get an error here.
	}
}

func TestServerInvalidHeader(t *testing.T) {
	t.Parallel()

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			if ctx.Request.Header.Peek("Foo") != nil || ctx.Request.Header.Peek("Foo ") != nil {
				t.Error("expected Foo header")
			}
		},
		Logger: &testLogger{},
	}

	ln := fasthttputil.NewInmemoryListener()

	go func() {
		if err := s.Serve(ln); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	c, err := ln.Dial()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err = c.Write([]byte("POST /foo HTTP/1.1\r\nHost: gle.com\r\nFoo : bar\r\nContent-Length: 5\r\n\r\n12345")); err != nil {
		t.Fatal(err)
	}

	br := bufio.NewReader(c)
	var resp fasthttp.Response
	if err := resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusBadRequest)
	}

	c, err = ln.Dial()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err = c.Write([]byte("GET /foo HTTP/1.1\r\nHost: gle.com\r\nFoo : bar\r\n\r\n")); err != nil {
		t.Fatal(err)
	}

	br = bufio.NewReader(c)
	if err := resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusBadRequest)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ln.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServerDisableKeepalive(t *testing.T) {
	t.Parallel()

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("OK") //nolint:errcheck
		},
		DisableKeepalive: true,
	}

	ln := fasthttputil.NewInmemoryListener()

	serverCh := make(chan struct{})
	go func() {
		if err := s.Serve(ln); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		close(serverCh)
	}()

	clientCh := make(chan struct{})
	go func() {
		c, err := ln.Dial()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if _, err = c.Write([]byte("GET / HTTP/1.1\r\nHost: aa\r\n\r\n")); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		br := bufio.NewReader(c)
		var resp fasthttp.Response
		if err = resp.Read(br); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if resp.StatusCode() != fasthttp.StatusOK {
			t.Errorf("unexpected status code: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusOK)
		}
		if !resp.ConnectionClose() {
			t.Error("expecting 'Connection: close' response header")
		}
		if string(resp.Body()) != "OK" {
			t.Errorf("unexpected body: %q. Expecting %q", resp.Body(), "OK")
		}

		// make sure the connection is closed
		data, err := ioutil.ReadAll(br)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(data) > 0 {
			t.Errorf("unexpected data read from the connection: %q. Expecting empty data", data)
		}

		close(clientCh)
	}()

	select {
	case <-clientCh:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-serverCh:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestServerTLSReadTimeout(t *testing.T) {
	t.Parallel()

	ln := fasthttputil.NewInmemoryListener()

	s := &fasthttp.Server{
		ReadTimeout: time.Millisecond * 500,
		//ReadTimeout: time.Second * 500,
		Logger: &testLogger{}, // Ignore log output.
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.SetBodyString("test response")
		},
	}

	certData, keyData, err := fasthttp.GenerateTestCertificate("localhost")
	if err != nil {
		t.Fatal(err)
	}

	err = s.AppendCertEmbed(certData, keyData)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		err = s.ServeTLS(ln, "", "")
		if err != nil {
			t.Error(err)
		}
	}()

	c, err := ln.Dial()
	if err != nil {
		t.Error(err)
	}

	r := make(chan error)

	go func() {
		b := make([]byte, 1)
		_, err := c.Read(b)
		c.Close()
		r <- err
	}()

	select {
	case err = <-r:
		t.Logf("TestServerTLSReadTimeout err: %s", err)
	case <-time.After(time.Second * 2):
	}

	if err == nil {
		t.Error("server didn't close connection after timeout")
	}
}

func TestTimeoutHandlerSuccess(t *testing.T) {
	t.Parallel()

	ln := fasthttputil.NewInmemoryListener()
	h := func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) == "/" {
			ctx.Success("aaa/bbb", []byte("real response"))
		}
	}
	s := &fasthttp.Server{
		Handler: fasthttp.TimeoutHandler(h, 10*time.Second, "timeout!!!"),
	}
	serverCh := make(chan struct{})
	go func() {
		if err := s.Serve(ln); err != nil {
			t.Errorf("unexepcted error: %v", err)
		}
		close(serverCh)
	}()

	concurrency := 20
	clientCh := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			conn, err := ln.Dial()
			if err != nil {
				t.Errorf("unexepcted error: %v", err)
			}
			if _, err = conn.Write([]byte("GET / HTTP/1.1\r\nHost: google.com\r\n\r\n")); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			br := bufio.NewReader(conn)
			verifyResponse(t, br, fasthttp.StatusOK, "aaa/bbb", "real response")
			clientCh <- struct{}{}
		}()
	}

	for i := 0; i < concurrency; i++ {
		select {
		case <-clientCh:
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-serverCh:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func verifyResponse(t *testing.T, r *bufio.Reader, expectedStatusCode int, expectedContentType, expectedBody string) *fasthttp.Response {
	var resp fasthttp.Response
	if err := resp.Read(r); err != nil {
		t.Fatalf("Unexpected error when parsing response: %v", err)
	}

	if !bytes.Equal(resp.Body(), []byte(expectedBody)) {
		t.Fatalf("Unexpected body %q. Expected %q", resp.Body(), []byte(expectedBody))
	}
	verifyResponseHeader(t, &resp.Header, expectedStatusCode, len(resp.Body()), expectedContentType, "")
	return &resp
}

func verifyResponseHeader(t *testing.T, h *fasthttp.ResponseHeader, expectedStatusCode, expectedContentLength int, expectedContentType, expectedContentEncoding string) {
	if h.StatusCode() != expectedStatusCode {
		t.Fatalf("Unexpected status code %d. Expected %d", h.StatusCode(), expectedStatusCode)
	}
	if h.ContentLength() != expectedContentLength {
		t.Fatalf("Unexpected content length %d. Expected %d", h.ContentLength(), expectedContentLength)
	}
	if string(h.ContentType()) != expectedContentType {
		t.Fatalf("Unexpected content type %q. Expected %q", h.ContentType(), expectedContentType)
	}
	if string(h.ContentEncoding()) != expectedContentEncoding {
		t.Fatalf("Unexpected content encoding %q. Expected %q", h.ContentEncoding(), expectedContentEncoding)
	}
}

func TestMaxReadTimeoutPerRequest(t *testing.T) {
	t.Parallel()

	headers := []byte(fmt.Sprintf("POST /foo2 HTTP/1.1\r\nHost: aaa.com\r\nContent-Length: %d\r\nContent-Type: aa\r\n\r\n", 5*1024))
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			t.Error("shouldn't reach handler")
		},
		HeaderReceived: func(header *fasthttp.RequestHeader) fasthttp.RequestConfig {
			return fasthttp.RequestConfig{
				ReadTimeout: time.Millisecond,
			}
		},
		ReadBufferSize: len(headers),
		ReadTimeout:    time.Second * 5,
		WriteTimeout:   time.Second * 5,
	}

	pipe := fasthttputil.NewPipeConns()
	cc, sc := pipe.Conn1(), pipe.Conn2()
	go func() {
		//write headers
		_, err := cc.Write(headers)
		if err != nil {
			t.Error(err)
		}
		//write body
		for i := 0; i < 5*1024; i++ {
			time.Sleep(time.Millisecond)
			cc.Write([]byte{'a'}) //nolint:errcheck
		}
	}()
	ch := make(chan error)
	go func() {
		ch <- s.ServeConn(sc)
	}()

	select {
	case err := <-ch:
		t.Logf("errCh: %s", err)
		if err == nil || err != nil && !strings.EqualFold(err.Error(), "timeout") {
			t.Fatalf("Unexpected error from serveConn: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("test timeout")
	}
}

func TestMaxWriteTimeoutPerRequest(t *testing.T) {
	t.Parallel()

	headers := []byte("GET /foo2 HTTP/1.1\r\nHost: aaa.com\r\nContent-Type: aa\r\n\r\n")
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
				var buf [192]byte
				for {
					w.Write(buf[:]) //nolint:errcheck
				}
			})
		},
		HeaderReceived: func(header *fasthttp.RequestHeader) fasthttp.RequestConfig {
			return fasthttp.RequestConfig{
				WriteTimeout: time.Millisecond,
			}
		},
		ReadBufferSize: 192,
		ReadTimeout:    time.Second * 5,
		WriteTimeout:   time.Second * 5,
	}

	pipe := fasthttputil.NewPipeConns()
	cc, sc := pipe.Conn1(), pipe.Conn2()

	var resp fasthttp.Response
	go func() {
		//write headers
		_, err := cc.Write(headers)
		if err != nil {
			t.Error(err)
		}
		br := bufio.NewReaderSize(cc, 192)
		err = resp.Header.Read(br)
		if err != nil {
			t.Error(err)
		}

		var chunk [192]byte
		for {
			time.Sleep(time.Millisecond)
			br.Read(chunk[:]) //nolint:errcheck
		}
	}()
	ch := make(chan error)
	go func() {
		ch <- s.ServeConn(sc)
	}()

	select {
	case err := <-ch:
		if err == nil || err != nil && !strings.EqualFold(err.Error(), "timeout") {
			t.Fatalf("Unexpected error from serveConn: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("test timeout")
	}
}

func TestIncompleteBodyReturnsUnexpectedEOF(t *testing.T) {
	t.Parallel()

	rw := &readWriter{}
	rw.r.WriteString("POST /foo HTTP/1.1\r\nHost: google.com\r\nContent-Length: 5\r\n\r\n123")
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {},
	}
	ch := make(chan error)
	go func() {
		ch <- s.ServeConn(rw)
	}()
	if err := <-ch; err == nil || err.Error() != "unexpected EOF" {
		t.Fatal(err)
	}
}

type readWriter struct {
	net.Conn
	r bytes.Buffer
	w bytes.Buffer
}

func (rw *readWriter) Close() error {
	return nil
}

func (rw *readWriter) Read(b []byte) (int, error) {
	return rw.r.Read(b)
}

func (rw *readWriter) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}

func (rw *readWriter) RemoteAddr() net.Addr {
	return zeroTCPAddr
}

func (rw *readWriter) LocalAddr() net.Addr {
	return zeroTCPAddr
}

func (rw *readWriter) SetDeadline(t time.Time) error {
	return nil
}

func (rw *readWriter) SetReadDeadline(t time.Time) error {
	return nil
}

func (rw *readWriter) SetWriteDeadline(t time.Time) error {
	return nil
}
