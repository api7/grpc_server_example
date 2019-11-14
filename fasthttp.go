package main

import (
    "github.com/valyala/fasthttp"
    "log"
    "runtime"
)

func main() {

    runtime.GOMAXPROCS(1)

    // The server will listen for incoming requests on this address.
    listenAddr := "127.0.0.1:1222"

    // This function will be called by the server for each incoming request.
    //
    // RequestCtx provides a lot of functionality related to http request
    // processing. See RequestCtx docs for details.
    requestHandler := func(ctx *fasthttp.RequestCtx) {
        getValues := ctx.QueryArgs()
        name := getValues.Peek("name")
        ctx.Write(name)

        //log.Printf("Received: %s ", name)
    }

    // Start the server with default settings.
    // Create Server instance for adjusting server settings.
    //
    // ListenAndServe returns only on error, so usually it blocks forever.
    if err := fasthttp.ListenAndServe(listenAddr, requestHandler); err != nil {
        log.Fatalf("error in ListenAndServe: %s", err)
    }
}
