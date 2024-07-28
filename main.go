package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"buf.build/gen/go/jyapp/runndemo/connectrpc/go/greet/v1/greetv1connect"
	"buf.build/gen/go/jyapp/runndemo/connectrpc/go/hello/v1/hellov1connect"
	"buf.build/gen/go/jyapp/runndemo/connectrpc/go/pubsub/v1/pubsubv1connect"
	greetv1 "buf.build/gen/go/jyapp/runndemo/protocolbuffers/go/greet/v1"
	hellov1 "buf.build/gen/go/jyapp/runndemo/protocolbuffers/go/hello/v1"
	pubsubv1 "buf.build/gen/go/jyapp/runndemo/protocolbuffers/go/pubsub/v1"
	"connectrpc.com/connect"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	greetChannel = "greet"
	helloChannel = "hello"
)

type GreetServer struct {
	redisClient *redis.Client
}

func (g *GreetServer) Greet(
	ctx context.Context,
	req *connect.Request[greetv1.GreetRequest],
) (*connect.Response[greetv1.GreetResponse], error) {
	fmt.Println("call greet")

	msg := fmt.Sprintf("Hi, %s", req.Msg.Message)
	if err := g.redisClient.Publish(ctx, greetChannel, msg).Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&greetv1.GreetResponse{}), nil
}

type HelloServer struct {
	redisClient *redis.Client
}

func (h *HelloServer) Hello(
	ctx context.Context,
	req *connect.Request[hellov1.HelloRequest],
) (*connect.Response[hellov1.HelloResponse], error) {
	fmt.Println("call hello")

	msg := fmt.Sprintf("Hello, %s", req.Msg.Message)
	if err := h.redisClient.Publish(ctx, helloChannel, msg).Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&hellov1.HelloResponse{}), nil
}

type PubSubServer struct {
	redisClient *redis.Client
}

func (p *PubSubServer) Subscribe(
	ctx context.Context,
	req *connect.Request[pubsubv1.SubscribeRequest],
	stream *connect.ServerStream[pubsubv1.SubscribeResponse],
) error {
	fmt.Println("call pubsub")

	pubsub := p.redisClient.Subscribe(ctx, greetChannel, helloChannel)
	defer func() {
		_ = pubsub.Close()
	}()

	ch := pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			fmt.Printf("channel: %s payload: %s\n", msg.Channel, msg.Payload)
			if err := stream.Send(&pubsubv1.SubscribeResponse{
				Message: msg.Payload,
			}); err != nil {
				return connect.NewError(connect.CodeInternal, err)
			}
		case <-ctx.Done():
			fmt.Println("context canceled")
			return nil
		}
	}
}

func main() {
	rdb := redis.NewClient(&redis.Options{})
	defer func() {
		_ = rdb.Close()
	}()

	var (
		greet  = &GreetServer{redisClient: rdb}
		hello  = &HelloServer{redisClient: rdb}
		pubsub = &PubSubServer{redisClient: rdb}
	)

	mux := http.NewServeMux()

	path, handler := greetv1connect.NewGreetServiceHandler(greet)
	mux.Handle(path, handler)

	path, handler = hellov1connect.NewHelloServiceHandler(hello)
	mux.Handle(path, handler)

	path, handler = pubsubv1connect.NewPubSubServiceHandler(pubsub)
	mux.Handle(path, handler)

	server := &http.Server{
		Addr:    "localhost:8080",
		Handler: h2c.NewHandler(mux, &http2.Server{}),
		// Don't forget timeouts!
	}

	go func() {
		fmt.Println("start connect server...")
		log.Fatal(server.ListenAndServe())
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit

	fmt.Println("shutdown server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	fmt.Println("end connect server...")
}
