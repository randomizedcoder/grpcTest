// from:
// https://github.com/grpc/grpc-go/blob/master/examples/features/retry/client/main.go
/*
 *
 * Copyright 2019 gRPC authors.
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

// Binary client demonstrates how to enable and configure retry policies for
// gRPC requests.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/features/proto/echo"
	"google.golang.org/grpc/metadata"
)

var (
	loops       = flag.Int("loops", 10, "loops")
	addr        = flag.String("addr", "localhost:50052", "the address to connect to")
	policy      = flag.String("policy", "grpc_client_policy.yaml", "filename of the grpc client policy.yaml")
	failpercent = flag.Int("failpercent", 50, "failpercent integers only between 0-100")
	failcodes   = flag.String("failcodes", "4,8,14", "failcodes header to insert. single code, or comma seperated")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	// https://github.com/grpc/grpc/blob/master/doc/service_config.md to know more about service config
	// https://github.com/grpc/grpc-go/blob/11feb0a9afd8/examples/features/retry/client/main.go#L36
	// https://grpc.github.io/grpc/core/md_doc_statuscodes.html
	servicePolicyBytes, err := os.ReadFile(*policy)
	if err != nil {
		log.Fatal(err)
	}

	// Set up a connection to the server with service config and create the channel.
	// However, the recommended approach is to fetch the retry configuration
	// (which is part of the service config) from the name resolver rather than
	// defining it on the client side.
	conn, err := grpc.NewClient(
		*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(string(servicePolicyBytes)),
		grpc.WithUnaryInterceptor(unaryClientInterceptor(*failpercent, *failcodes)),
	)

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func() {
		if e := conn.Close(); e != nil {
			log.Printf("failed to close connection: %s", e)
		}
	}()

	c := pb.NewEchoClient(conn)
	for i := 0; i < *loops; i++ {

		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		reply, err := c.UnaryEcho(ctx,
			&pb.EchoRequest{
				Message: "Try and Success",
			},
		)
		if err != nil {
			log.Printf("i:%d UnaryEcho error: %v", i, err)
		}
		log.Printf("i:%d UnaryEcho reply: %v", i, reply)
	}
}

func unaryClientInterceptor(failpercent int, failcodes string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		md := metadata.Pairs(
			"failpercent", fmt.Sprintf("%d", failpercent),
			"failcodes", failcodes,
		)
		ctxMD := metadata.NewOutgoingContext(ctx, md)

		return invoker(ctxMD, method, req, reply, cc, opts...)
	}
}
