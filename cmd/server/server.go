// from:
// https://github.com/grpc/grpc-go/blob/master/examples/features/retry/server/main.go
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

// Binary server demonstrates to enforce retries on client side.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	_ "unsafe"

	pb "google.golang.org/grpc/examples/features/proto/echo"
)

// unsafe for FastRandN

// https://cs.opensource.google/go/go/+/master:src/runtime/stubs.go;l=151?q=FastRandN&ss=go%2Fgo
// https://lemire.me/blog/2016/06/27/a-fast-alternative-to-the-modulo-reduction/

//go:linkname FastRandN runtime.fastrandn
func FastRandN(n uint32) uint32

const (
	// https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
	// We want a range between 1 and 16, so our maxCode is 15, cos we will +1
	maxCode = 15
)

var (
	success atomic.Uint64
	fail    atomic.Uint64

	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
)

type echoServer struct {
	pb.UnimplementedEchoServer
}

func newEchoServer() (s echoServer) {
	s = *new(echoServer)
	return s
}

func (s echoServer) UnaryEcho(_ context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {

	log.Println("request succeeded count:", success.Add(1))

	return &pb.EchoResponse{Message: req.Message}, nil
}

// https://pkg.go.dev/google.golang.org/grpc?utm_source=godoc#UnaryServerInterceptor
func unaryServerInterceptor(failureRate int) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

		// https://grpc.io/docs/guides/metadata/
		// https://github.com/grpc/grpc-go/blob/master/examples/features/metadata/server/main.go
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, errMissingMetadata
		}

		fp, errG := readFailPercent(failureRate, &md)
		if errG != nil {
			return nil, errG
		}

		fcs, errC := readFailCodes(&md)
		if errC != nil {
			return nil, errC
		}

		rp := FastRandN(100)
		if rp < uint32(fp) {

			f := fail.Add(1)

			var code codes.Code

			switch len(fcs) {
			case 0:
				code = anyRandomCode()
			case 1:
				code = fcs[0]
			default:
				code = returnAnySuppliedCode(&fcs)
			}

			log.Printf("request failed count:%d code:%s", f, code.String())

			return nil, status.Errorf(code, "intercept failure code:%d rp:%d fail:%d", uint32(code), rp, f)
		}

		return handler(ctx, req)
	}
}

// readFailPercent reads the "failpercent" percentage, including validation
// percentage needs to be a integer between 0-100
// e.g. failme = 10 ( 10% )
// e.g. failme = 90 ( 90% )
func readFailPercent(failureRate int, md *metadata.MD) (int, error) {

	// metadata keys are always lower case
	// https://github.com/grpc/grpc-go/blob/v1.68.0/metadata/metadata.go#L207
	if t, ok := (*md)["failpercent"]; ok {
		i, err := strconv.ParseInt(t[0], 0, 64)
		if err != nil {
			return 0, status.Error(codes.InvalidArgument, "failme ParseInt error")
		}
		errV := validateFailurePercent(int(i))
		if errV != nil {
			return 0, status.Error(codes.InvalidArgument, "failme validateFailurePercent error")
		}
		log.Printf("failme:%d from metadata:\n", i)

		return int(i), nil
	}
	return failureRate, nil
}

// validateFailurePercent ensure the percentage is between 0-100 inclusive
func validateFailurePercent(failureRate int) error {
	if failureRate < 0 || failureRate > 100 {
		return errors.New("invalid failureRate")
	}
	return nil
}

// readFailCodes returns a slice of codes.Code from the metadata "failcodes"
// calls can include a single code, or a comma seperated list of codes
// e.g. failcodes = 14 (unavailable)
// e.g. failcodes = 10,12,14
// valid codes: https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
func readFailCodes(md *metadata.MD) (cs []codes.Code, errR error) {

	if fc, ok := (*md)["failcodes"]; ok {

		if !strings.Contains(fc[0], ",") {
			c, err := strconv.ParseInt(fc[0], 0, 64)
			if err != nil {
				return cs, status.Error(codes.InvalidArgument, "failcodes ParseInt error")
			}
			if validateCodeUint32(uint32(c)) != nil {
				return cs, status.Error(codes.InvalidArgument, "failcodes validate error")
			}
			cs = append(cs, codes.Code(uint32(c)))
			return cs, nil
		}

		parts := strings.Split(fc[0], ",")
		for i := 0; i < len(parts); i++ {
			i, err := strconv.ParseInt(parts[i], 0, 64)
			if err != nil {
				return cs, status.Error(codes.InvalidArgument, "failcodes ParseInt error")
			}
			if validateCodeUint32(uint32(i)) != nil {
				return cs, status.Error(codes.InvalidArgument, "failcodes validate error")
			}
			cs = append(cs, codes.Code(uint32(i)))
		}
		return cs, nil
	}
	return cs, nil
}

// validateCodeUint32 ensure the code is between 0-16 inclusive
// code can't be < 0 because it's a uint32
func validateCodeUint32(code uint32) error {
	if code > 16 {
		return errors.New("invalid code")
	}
	return nil
}

// anyRandomCode returns ANY random valid code ( 0-16 )
// if the request metadata does not contain "failcodes"
func anyRandomCode() (code codes.Code) {
	return codes.Code(FastRandN(maxCode) + 1)
}

// returnAnySuppliedCode randomly selects from the
// metadata supplied list of codes in "failcodes"
func returnAnySuppliedCode(cs *[]codes.Code) (code codes.Code) {
	return (*cs)[int(FastRandN(uint32(len(*cs))))]
}

func main() {

	port := flag.Int("port", 50052, "port number")
	address := fmt.Sprintf(":%v", *port)
	failurePercent := flag.Int("failurePercent", 50, "failurePercent")

	flag.Parse()

	lis, err := net.Listen("tcp", address)

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("listen on address", address)

	s := grpc.NewServer(
		grpc.UnaryInterceptor(unaryServerInterceptor(*failurePercent)),
	)

	srv := newEchoServer()

	pb.RegisterEchoServer(s, srv)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
