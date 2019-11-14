package main

import (
  "log"
  "net"
  "sync"
  "time"

  "github.com/mwitkow/grpc-proxy/proxy"
  "golang.org/x/net/context"
  "google.golang.org/grpc"
  "google.golang.org/grpc/connectivity"
  "google.golang.org/grpc/metadata"
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

var (
  director proxy.StreamDirector
)

func main() {

  p := NewPool(100, time.Minute)

  director = func(ctx context.Context, fmn string) (context.Context, *grpc.ClientConn, error) {
    //func myDirector(ctx context.Context, fmn string) (*grpc.ClientConn, error) {

    md, _ := metadata.FromIncomingContext(ctx)
    // Copy the inbound metadata explicitly.
    outCtx, _ := context.WithCancel(ctx)
    outCtx = metadata.NewOutgoingContext(outCtx, md.Copy())

    conn, err := p.Get("127.0.0.1:50051", grpc.WithCodec(proxy.Codec()), grpc.WithInsecure())
    if err != nil {
      log.Fatalf("did not connect: %v", err)
    }

    defer p.Put("127.0.0.1:50051", conn, nil)

    // conn, err := grpc.DialContext(ctx, "127.0.0.1:50051",
    //   grpc.WithCodec(proxy.Codec()),
    //   grpc.WithInsecure())

    return outCtx, conn.GetCC(), err
  }

  l, err := net.Listen("tcp", "127.0.0.1:8888")
  if err != nil {
    log.Fatalln(err)
  }
  s := grpc.NewServer(
    grpc.CustomCodec(proxy.Codec()),
    grpc.UnknownServiceHandler(proxy.TransparentHandler(director)))
  if err := s.Serve(l); err != nil {
    log.Fatalln(err)
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
