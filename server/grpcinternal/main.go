package main

import (
	"context"
	"net"
	"net/http"
	"os"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/komem3/checktrace"
	"github.com/komem3/checktrace/internal/gcp"
	"github.com/komem3/checktrace/internal/logger"
	"github.com/komem3/checktrace/protogen"
	"github.com/rs/zerolog"
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

	grpcPort := "9000"

	go runGrpc(project, &log, grpcPort)
	runGateWay(ctx, log, "localhost:"+grpcPort)
}

func runGateWay(ctx context.Context, log zerolog.Logger, target string) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	conn, err := grpc.DialContext(ctx, target,
		grpc.WithInsecure(),
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
	)
	if err != nil {
		log.Panic().Err(err).Msg("prepare dial")
	}
	log.Info().Str("target", target).Msg("connection")

	if err = protogen.RegisterTraceServiceHandler(ctx, mux, conn); err != nil {
		log.Panic().Err(err).Msg("register mux")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	httpHandler := &ochttp.Handler{
		Handler: mux,
		// Use the Google Cloud propagation format.
		Propagation: &propagation.HTTPFormat{},
	}
	log.Info().Str("port", port).Msg("start gateway")
	err = http.ListenAndServe(":"+port, httpHandler)
	log.Panic().Err(err).Msg("http serve")
}

func runGrpc(project string, log *zerolog.Logger, port string) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Panic().Err(err).Str("port", port).Msg("listen tcp")
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			logger.LogInterceptor(project, *log),
		)),
	)
	protogen.RegisterTraceServiceServer(grpcServer, &checktrace.TraceService{})

	log.Info().Str("port", port).Msg("start grpc")
	err = grpcServer.Serve(lis)
	log.Panic().Err(err).Msg("start grpc server")
}
