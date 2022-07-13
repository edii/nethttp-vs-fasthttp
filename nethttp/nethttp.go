package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", HelloServer)
	server := &http.Server{
		Addr:              ":1234",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
	}
	server.ListenAndServe()
}

//func HelloServer(w http.ResponseWriter, r *http.Request) {
//	fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
//}

func HelloServer(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "serving endpoint %v\n", r.URL.Path[1:])
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "while reading body: %v", err)
	} else {
		fmt.Fprintf(w, "Hello, %s!", string(body))
	}
}
