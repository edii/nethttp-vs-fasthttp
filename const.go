package httpclient

import "time"

const (
	WriteTimeout = 1 * time.Second
	ReadTimeout  = 1 * time.Second
	IdleTimeout  = 30 * time.Second
)
