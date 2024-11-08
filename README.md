# grpcTest

Proof of concept code to use interceptors on both the GRPC client and server

## Client usage
The client can make requests with "failpercent" and "failcode" metadata(headers)

"failpercent"
percentage needs to be a integer between 0-100
e.g. failme = 10 ( 10% )
e.g. failme = 90 ( 90% )

"failcodes"
a single code, or a comma seperated list of codes
// e.g. failcodes = 14 (unavailable)
// e.g. failcodes = 10,12,14

If "failcodes" is NOT supplied, any random code is returned as the error

Possible failcodes are:
https://github.com/grpc/grpc/blob/master/doc/statuscodes.md


## Example

failpercent = 100 ( all requests will fail)
failcodes is not specified, so we get any random status code
```
[das@t:~/Downloads/grpcTest/cmd/client]$ ./client --loops 5 --failpercent 100
2024/11/07 19:54:02 i:0 UnaryEcho error: rpc error: code = ResourceExhausted desc = intercept failure code:8 rp:41 fail:51
2024/11/07 19:54:02 i:0 UnaryEcho reply: <nil>
2024/11/07 19:54:02 i:1 UnaryEcho error: rpc error: code = Unavailable desc = intercept failure code:14 rp:42 fail:52
2024/11/07 19:54:02 i:1 UnaryEcho reply: <nil>
2024/11/07 19:54:02 i:2 UnaryEcho error: rpc error: code = DeadlineExceeded desc = intercept failure code:4 rp:16 fail:53
2024/11/07 19:54:02 i:2 UnaryEcho reply: <nil>
2024/11/07 19:54:02 i:3 UnaryEcho error: rpc error: code = DeadlineExceeded desc = intercept failure code:4 rp:49 fail:54
2024/11/07 19:54:02 i:3 UnaryEcho reply: <nil>
2024/11/07 19:54:02 i:4 UnaryEcho error: rpc error: code = Unavailable desc = intercept failure code:14 rp:13 fail:55
2024/11/07 19:54:02 i:4 UnaryEcho reply: <nil>
```

failpercent = 100 ( all requests will fail)
failcodes = 14 (unavailable), so the server will only return code 14
```
[das@t:~/Downloads/grpcTest/cmd/client]$ ./client --loops 5 --failpercent 100 --failcodes 14
2024/11/07 19:54:07 i:0 UnaryEcho error: rpc error: code = Unavailable desc = intercept failure code:14 rp:96 fail:56
2024/11/07 19:54:07 i:0 UnaryEcho reply: <nil>
2024/11/07 19:54:07 i:1 UnaryEcho error: rpc error: code = Unavailable desc = intercept failure code:14 rp:61 fail:57
2024/11/07 19:54:07 i:1 UnaryEcho reply: <nil>
2024/11/07 19:54:07 i:2 UnaryEcho error: rpc error: code = Unavailable desc = intercept failure code:14 rp:48 fail:58
2024/11/07 19:54:07 i:2 UnaryEcho reply: <nil>
2024/11/07 19:54:07 i:3 UnaryEcho error: rpc error: code = Unavailable desc = intercept failure code:14 rp:26 fail:59
2024/11/07 19:54:07 i:3 UnaryEcho reply: <nil>
2024/11/07 19:54:07 i:4 UnaryEcho error: rpc error: code = Unavailable desc = intercept failure code:14 rp:90 fail:60
2024/11/07 19:54:07 i:4 UnaryEcho reply: <nil>

failpercent = 100 ( all requests will fail)
failcodes =  10,12,14, so the server will randomly return one of the codes 10, 12, or 14
```
[das@t:~/Downloads/grpcTest/cmd/client]$ ./client --loops 5 --failpercent 100 --failcodes 10,12,14
2024/11/07 19:54:10 i:0 UnaryEcho error: rpc error: code = Aborted desc = intercept failure code:10 rp:63 fail:61
2024/11/07 19:54:10 i:0 UnaryEcho reply: <nil>
2024/11/07 19:54:10 i:1 UnaryEcho error: rpc error: code = Unimplemented desc = intercept failure code:12 rp:39 fail:62
2024/11/07 19:54:10 i:1 UnaryEcho reply: <nil>
2024/11/07 19:54:10 i:2 UnaryEcho error: rpc error: code = Unavailable desc = intercept failure code:14 rp:24 fail:63
2024/11/07 19:54:10 i:2 UnaryEcho reply: <nil>
2024/11/07 19:54:10 i:3 UnaryEcho error: rpc error: code = Unavailable desc = intercept failure code:14 rp:59 fail:64
2024/11/07 19:54:10 i:3 UnaryEcho reply: <nil>
2024/11/07 19:54:10 i:4 UnaryEcho error: rpc error: code = Aborted desc = intercept failure code:10 rp:60 fail:65
2024/11/07 19:54:10 i:4 UnaryEcho reply: <nil>
```