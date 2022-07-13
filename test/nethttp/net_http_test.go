package nethttp

import (
	"net"
	"testing"
	"time"

	"net/http"
)

func TestNetHttpConnect(t *testing.T) {
	t.Parallel()

	client := http.Client{
		Timeout: time.Second,
	}
	start := time.Now()
	_, err := client.Get(ConnectUrl())
	assertTimeout(t, start, err, "Client.Timeout exceeded")
}

func TestNetHttpRead(t *testing.T) {
	t.Parallel()

	client := http.Client{
		Timeout: time.Second,
	}
	start := time.Now()
	_, err := client.Get(ReadUrl())
	assertTimeout(t, start, err, "Client.Timeout exceeded")
}

func TestNetConnect(t *testing.T) {
	t.Parallel()

	start := time.Now()
	_, err := net.DialTimeout("tcp", ConnectHostAndPort(), time.Second)
	assertTimeout(t, start, err, "i/o timeout")
}

func TestNetRead(t *testing.T) {
	t.Parallel()

	conn, err := net.Dial("tcp", ReadHostAndPort())
	if err != nil {
		panic(err)
	}
	err = conn.SetDeadline(time.Now().Add(time.Second))
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 1024)
	start := time.Now()
	_, err = conn.Read(buf)
	assertTimeout(t, start, err, "i/o timeout")
}
