package cloudtrace

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"

	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const traceKey = "X-Cloud-Trace-Context"

func TraceServerInterceptor(name string, option ...trace.StartOption) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, _ error) {
		meta, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}
		spanCtx, ok := (&propagation.HTTPFormat{}).SpanContextFromRequest(&http.Request{
			Header: http.Header{
				traceKey: meta.Get(traceKey),
			},
		})
		if !ok {
			return handler(ctx, req)
		}
		ctx, _ = trace.StartSpanWithRemoteParent(ctx, name, spanCtx, option...)
		return handler(ctx, req)
	}
}

func TraceClientInterceptor(name string, option ...trace.StartOption) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		sc := trace.FromContext(ctx).SpanContext()
		sid := binary.BigEndian.Uint64(sc.SpanID[:])
		header := fmt.Sprintf("%s/%d;o=%d", hex.EncodeToString(sc.TraceID[:]), sid, int64(sc.TraceOptions))
		ctx = metadata.AppendToOutgoingContext(ctx, traceKey, header)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
