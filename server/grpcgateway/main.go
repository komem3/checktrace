package main

import (
	"context"
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/komem3/checktrace/internal/cloudtrace"
	"github.com/komem3/checktrace/internal/gcp"
	"github.com/komem3/checktrace/internal/logger"
	"github.com/komem3/checktrace/protogen"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()
	log := logger.NewLogger()

	project, err := gcp.Project("test")
	if err != nil {
		log.Panic().Err(err).Msg("")
	}
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: project,
	})
	if err != nil {
		log.Panic().Err(err).Msg("")
	}
	trace.RegisterExporter(exporter)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	target := os.Getenv("TARGET")
	targetPort := os.Getenv("TARGET_PORT")

	var chainInterceptor grpc.DialOption
	if metadata.OnGCE() {
		chainInterceptor = grpc.WithChainUnaryInterceptor(
			gcp.AuthServiceUnnaryClientInterceptor(target),
			cloudtrace.TraceClientInterceptor(project),
		)
	} else {
		chainInterceptor = grpc.WithChainUnaryInterceptor(
			cloudtrace.TraceClientInterceptor(project),
		)
	}

	conn, err := gcp.NewGrpcConnection(ctx, target+":"+targetPort, !metadata.OnGCE(),
		chainInterceptor,
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
	)
	if err != nil {
		log.Panic().Err(err).Msg("new connection option")
	}
	log.Info().Str("target", target).Msg("connection")

	mux := runtime.NewServeMux()

	if err = protogen.RegisterTraceServiceHandler(ctx, mux, conn); err != nil {
		log.Panic().Err(err).Msg("register mux")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}

	httpHandler := &ochttp.Handler{
		Handler: mux,
		// Use the Google Cloud propagation format.
		Propagation: &propagation.HTTPFormat{},
	}
	log.Info().Str("port", port).Msg("start gateway")
	log.Panic().Err(http.ListenAndServe(":"+port, httpHandler))
}
