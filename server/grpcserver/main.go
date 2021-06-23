package main

import (
	"net"
	"os"

	"contrib.go.opencensus.io/exporter/stackdriver"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/komem3/checktrace"
	"github.com/komem3/checktrace/internal/cloudtrace"
	"github.com/komem3/checktrace/internal/gcp"
	"github.com/komem3/checktrace/internal/logger"
	"github.com/komem3/checktrace/protogen"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
)

func main() {
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Panic().Err(err).Str("port", port).Msg("listen tcp")
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			cloudtrace.TraceServerInterceptor(project),
			logger.LogInterceptor(project, log),
		)),
	)
	protogen.RegisterTraceServiceServer(grpcServer, &checktrace.TraceService{})
	log.Info().Str("port", port).Msg("start grpc")
	log.Panic().Err(grpcServer.Serve(lis))
}
