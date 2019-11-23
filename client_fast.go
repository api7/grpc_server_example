/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a client for Greeter service.
package main

import (
    "context"
    "encoding/json"
    "log"
    "runtime"
    "sync"
    "time"

    "github.com/valyala/fasthttp"

    "google.golang.org/grpc"
    "google.golang.org/grpc/connectivity"

    pb "github.com/nic-chen/grpc_server_example/proto"
)

type Pool struct {
    size int
    ttl  int64

    sync.Mutex
    conns map[string][]*poolConn
}

type poolConn struct {
    cc      *grpc.ClientConn
    created int64
}

const (
    address = "127.0.0.1:50051"
)

func main() {

    runtime.GOMAXPROCS(1)

    p := NewPool(300, time.Minute)

    listenAddr := "127.0.0.1:1212"

    requestHandler := func(ctx *fasthttp.RequestCtx) {
        getValues := ctx.QueryArgs()
        name := getValues.Peek("name")

        conn, err := p.Get(address, grpc.WithInsecure())
        if err != nil {
            log.Fatalf("did not connect: %v", err)
        }

        defer p.Put(address, conn, nil)

        c := pb.NewGreeterClient(conn.GetCC())

        //ctx2, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

        clientDeadline := time.Now().Add(time.Duration(100) * time.Millisecond)
        ctx2, cancel := context.WithDeadline(ctx, clientDeadline)

        defer cancel()
        gr, err := c.SayHello(ctx2, &pb.HelloRequest{Name: string(name)})

        if err != nil {
            log.Fatalf("could not greet: %v", err)
        }

        m1 := make(map[string]interface{})
        m1["message"] = gr.GetMessage()

        b4, err := json.Marshal(m1)

        ctx.Write([]byte(b4))
    }

    if err := fasthttp.ListenAndServe(listenAddr, requestHandler); err != nil {
        log.Fatalf("error in ListenAndServe: %s", err)
    }

}

func NewPool(size int, ttl time.Duration) *Pool {
    return &Pool{
        size:  size,
        ttl:   int64(ttl.Seconds()),
        conns: make(map[string][]*poolConn),
    }
}

func (p *Pool) Get(addr string, opts ...grpc.DialOption) (*poolConn, error) {
    p.Lock()
    conns := p.conns[addr]
    now := time.Now().Unix()

    // while we have conns check age and then return one
    // otherwise we'll create a new conn
    for len(conns) > 0 {
        conn := conns[len(conns)-1]
        conns = conns[:len(conns)-1]
        p.conns[addr] = conns

        // if conn is old or not ready kill it and move on
        if d := now - conn.created; d > p.ttl || conn.cc.GetState() != connectivity.Ready {
            conn.cc.Close()
            continue
        }

        // we got a good conn, lets unlock and return it
        p.Unlock()

        return conn, nil
    }

    p.Unlock()

    // create new conn
    cc, err := grpc.Dial(addr, opts...)
    if err != nil {
        return nil, err
    }

    return &poolConn{cc, time.Now().Unix()}, nil
}

func (p *Pool) Put(addr string, conn *poolConn, err error) {
    // don't store the conn if it has errored
    if err != nil {
        conn.cc.Close()
        return
    }

    // otherwise put it back for reuse
    p.Lock()
    conns := p.conns[addr]
    if len(conns) >= p.size {
        p.Unlock()
        conn.cc.Close()
        return
    }
    p.conns[addr] = append(conns, conn)
    p.Unlock()
}

func (pc *poolConn) GetCC() *grpc.ClientConn {
    return pc.cc
}
