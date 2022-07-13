package main

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"time"
)

func FastHttpServ() {
	srv := &fasthttp.Server{
		ReadTimeout: 10 * time.Second,
		Handler:     HelloServer,
	}

	if err := srv.ListenAndServe(":1234"); err != nil {
		fmt.Printf("Server failed: %s\n", err)
	}
	defer srv.Shutdown()
}

func main() {
	FastHttpServ()
}

func HelloServer(ctx *fasthttp.RequestCtx) {
	ctx.SetBodyString(fmt.Sprintf("Hello, %s!", string(ctx.Request.Body())))
}
