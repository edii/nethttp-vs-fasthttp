package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func slowHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Call Handler!!!")
	time.Sleep(2 * time.Second)
	io.WriteString(w, "I am slow!\n")
}

func HttpServ() {
	srv := http.Server{
		//ReadTimeout:  httpclient.ReadTimeout,
		//WriteTimeout: 10 * time.Microsecond,
		WriteTimeout: 10 * time.Second,
		//IdleTimeout:  httpclient.IdleTimeout,
		Addr:    ":8888",
		Handler: http.HandlerFunc(slowHandler),
	}

	if err := srv.ListenAndServe(); err != nil {
		fmt.Printf("Server failed: %s\n", err)
	}
}

func main() {
	HttpServ()
}
