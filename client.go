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
    "net/http"
    "os"
    "runtime"
    "time"

    pb "github.com/nic-chen/grpc_server_example/proto"
    "google.golang.org/grpc"
)

const (
    address = "127.0.0.1:50051"
)

func main() {

    runtime.GOMAXPROCS(1)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        r.ParseForm()
        name := r.Form.Get("name")

        // Set up a connection to the server.
        conn, err := grpc.Dial(address, grpc.WithInsecure())
        if err != nil {
            log.Fatalf("did not connect: %v", err)
        }
        defer conn.Close()
        c := pb.NewGreeterClient(conn)

        // Contact the server and print out its response.
        if len(os.Args) > 1 {
            name = os.Args[1]
        }
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()
        gr, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
        if err != nil {
            log.Fatalf("could not greet: %v", err)
        }

        m1 := make(map[string]interface{})
        m1["message"] = gr.GetMessage()

        b4, err := json.Marshal(m1)

        w.Write([]byte(b4))
    })

    http.ListenAndServe(":1210", nil)
}
