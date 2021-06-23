package gcp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// NewConn creates a new gRPC connection.
// host should be of the form domain:port, e.g., example.com:443
func NewGrpcConnection(ctx context.Context, host string, insecure bool, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if host != "" {
		opts = append(opts, grpc.WithAuthority(host))
	}

	if insecure {
		opts = append(opts, grpc.WithInsecure())
	} else {
		systemRoots, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		cred := credentials.NewTLS(&tls.Config{
			RootCAs: systemRoots,
		})
		opts = append(opts, grpc.WithTransportCredentials(cred))
	}

	return grpc.DialContext(ctx, host, opts...)
}

func AuthServiceUnnaryClientInterceptor(audience string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Create an identity token.
		// With a global TokenSource tokens would be reused and auto-refreshed at need.
		// A given TokenSource is specific to the audience.
		tokenSource, err := idtoken.NewTokenSource(ctx, audience)
		if err != nil {
			return fmt.Errorf("idtoken.NewTokenSource: %v", err)
		}
		token, err := tokenSource.Token()
		if err != nil {
			return fmt.Errorf("TokenSource.Token: %v", err)
		}
		// Add token to gRPC Request.
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token.AccessToken)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
