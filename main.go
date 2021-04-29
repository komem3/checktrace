package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/komem3/checktrace/protogen"
	"github.com/rs/zerolog"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()
	logger := zerolog.New(os.Stderr).Hook(LevelNameHook{})

	project := "test"
	if metadata.OnGCE() {
		p, err := metadata.ProjectID()
		if err != nil {
			logger.Fatal().Err(err).Msg("")
		}
		project = p
	}
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: project,
	})
	if err != nil {
		log.Fatal(err)
	}
	trace.RegisterExporter(exporter)

	port := "9000"

	runGrpc(project, &logger, port)
	runGateWay(ctx, logger, "localhost:"+port)
}

func runGateWay(ctx context.Context, logger zerolog.Logger, target string) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	conn, err := grpc.DialContext(ctx, target,
		grpc.WithInsecure(),
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("prepare dial")
	}
	logger.Info().Str("target", target).Msg("connection")

	if err = protogen.RegisterTraceServiceHandler(ctx, mux, conn); err != nil {
		logger.Fatal().Err(err).Msg("register mux")
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
	logger.Info().Str("port", port).Msg("start gateway")
	err = http.ListenAndServe(":"+port, httpHandler)
	logger.Fatal().Err(err).Msg("http serve")
}

func runGrpc(project string, logger *zerolog.Logger, port string) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Fatal().Err(err).Str("port", port).Msg("listen tcp")
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			LogInterceptor(project, logger),
		)),
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
	)
	protogen.RegisterTraceServiceServer(grpcServer, &traceService{})

	logger.Info().Str("port", port).Msg("start grpc")
	go func() {
		err = grpcServer.Serve(lis)
		logger.Fatal().Err(err).Msg("start grpc server")
	}()
}
