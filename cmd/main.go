package main

import (
	"context"
	"github.com/pshvedko/mapslice"
	"github.com/pshvedko/mapslice/w2c"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/pshvedko/mapslice/gen/jerk/v1"
	"github.com/pshvedko/mapslice/gen/jerk/v1/jerkconnect"
)

type PingService struct {
	atomic.Uint64
	*mapslice.MapSlice[int32, uint64]
}

func (p *PingService) Trace(ctx context.Context, req *connect.Request[jerk.TraceRequest], out *connect.ServerStream[jerk.TraceResponse]) error {
	sub := p.Subscribe(req.Msg.GetKeys()...)
	defer p.Unsubscribe(sub)
	for {
		select {
		case <-sub.Ready():
			keys, indexes := sub.Load()
			for i, index := range indexes {
				err := out.Send(&jerk.TraceResponse{Key: keys[i], Indexes: index})
				if err != nil {
					return connect.NewError(connect.CodeInternal, err)
				}
			}
		case <-ctx.Done():
			return connect.NewWireError(connect.CodeAborted, http.ErrAbortHandler)
		}
	}
}

func (p *PingService) Ping(ctx context.Context, req *connect.Request[jerk.PingRequest]) (*connect.Response[jerk.PingResponse], error) {
	key := rand.Int31() % 100
	index := p.Add(1)
	p.Append(key, index)
	select {
	case <-time.After(time.Duration(req.Msg.GetTimeout()) * time.Millisecond):
	case <-ctx.Done():
	}
	res := connect.NewResponse(&jerk.PingResponse{Index: index})
	res.Header().Set("Ping-Version", "v1")
	return res, nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.TODO(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	mux := http.NewServeMux()
	compress := connect.WithCompressMinBytes(1024)
	service := &PingService{MapSlice: mapslice.NewMapSlice[int32, uint64]()}
	path, handler := jerkconnect.NewJerkServiceHandler(service, compress)
	mux.Handle(path, handler)
	names := []string{
		jerkconnect.JerkServiceName,
	}
	reflector := grpcreflect.NewStaticReflector(names...)
	mux.Handle(grpcreflect.NewHandlerV1(reflector, compress))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector, compress))
	health := grpchealth.NewStaticChecker(names...)
	mux.Handle(grpchealth.NewHandler(health, compress))

	w := w2c.NewHandler(h2c.NewHandler(mux, &http2.Server{}))
	defer w.Wait()

	s := http.Server{
		Addr:        "localhost:8080",
		Handler:     w,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	stop := context.AfterFunc(ctx, func() {
		w.Add(1)
		err := s.Shutdown(context.TODO())
		if err != nil {
			log.Println(err)
		}
		w.Done()
	})
	defer stop()

	err := s.ListenAndServe()
	if err != nil {
		log.Println(err)
	}
}
