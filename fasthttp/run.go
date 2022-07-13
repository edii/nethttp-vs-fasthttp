package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

func slowHandlerFastHTTP(ctx *fasthttp.RequestCtx) {
	buf := &bytes.Buffer{}
	wg := sync.WaitGroup{}

	defer timeTrack(time.Now(), "factorial")

	time.Sleep(2 * time.Second)

	data := &TestJson{
		Id:           "test-id",
		Name:         "this-name",
		Descriptions: "this-descriptions",
	}

	//time.Sleep(2 * time.Second)
	bData, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("err: %#v", err)
	}

	wg.Add(1)

	go func() {
		defer wg.Done()
		buf.Write(bData)
	}()

	wg.Wait()
	//log.Printf("resp: %+v", buf.String())

	//ctx.Response.SetBodyRaw(bData)
	ctx.Response.SetBodyRaw(buf.Bytes())
	//ctx.Response.SetBodyString("BBBB")
	ctx.Response.Header.SetContentType("application/json")
}

func FastHttpServ() {
	srv := &fasthttp.Server{
		//ReadTimeout: 1 * time.Second,
		//WriteTimeout: 10 * 1024 * time.Nanosecond,
		//WriteTimeout: 10 * time.Microsecond,
		WriteTimeout: 10 * time.Second,

		//ReadTimeout:  time.Second + 500*time.Millisecond,
		//WriteTimeout: 1 * time.Second,
		////MaxRequestBodySize: 2 * 1024,

		//ReadTimeout:  httpclient.ReadTimeout,
		//WriteTimeout: httpclient.WriteTimeout,
		//IdleTimeout:  httpclient.IdleTimeout,
		//Handler:      fasthttp.TimeoutHandler(slowHandlerFastHTTP, 2*time.Second, "Timeout!\n"),
		Handler: slowHandlerFastHTTP,
	}

	if err := srv.ListenAndServe(":8889"); err != nil {
		fmt.Printf("Server failed: %s\n", err)
	}
	defer srv.Shutdown()
}

type TestJson struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Descriptions string `json:"descriptions"`
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func main() {
	FastHttpServ()
}
