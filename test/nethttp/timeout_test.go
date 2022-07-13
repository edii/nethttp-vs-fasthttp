package nethttp

import (
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"time"
)

// timeoutConnection is an augmented connection with read/write timeouts that
// are used to set deadlines on each read/write operation.
type timeoutConnection struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// wrap the raw connection read with a deadline.
func (c *timeoutConnection) Read(b []byte) (int, error) {
	if c.ReadTimeout > 0 {
		err := c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
		if err != nil {
			return 0, err
		}
	}
	return c.Conn.Read(b)
}

// wrap the raw connection write with a deadline.
func (c *timeoutConnection) Write(b []byte) (int, error) {
	if c.WriteTimeout > 0 {
		err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
		if err != nil {
			return 0, err
		}
	}
	return c.Conn.Write(b)
}

func main() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte("Hello, client\n"))
	}))
	defer server.Close()

	// The net.Dialer settings were cribbed from http.DefaultTransport
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	dial := func(network, address string) (net.Conn, error) {
		conn, err := dialer.Dial(network, address)
		if err != nil {
			return nil, err
		}
		tc := &timeoutConnection{
			Conn:         conn,
			ReadTimeout:  time.Duration(100 * time.Millisecond),
			WriteTimeout: time.Duration(100 * time.Millisecond),
		}
		return tc, nil
	}
	transport := &http.Transport{
		Dial:                dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	req, _ := http.NewRequest("GET", server.URL, nil)

	// expect that our round trip has timed out.
	_, err := transport.RoundTrip(req)
	if err == nil {
		log.Fatal("expected a RoundTrip error")
	}
	neterr, ok := err.(net.Error)
	if !ok {
		log.Fatalf("expected a net.Error; got %#v", err)
	}
	if !neterr.Timeout() {
		log.Fatalf("expected a timeout error")
	}
	log.Print("Success.")
}
