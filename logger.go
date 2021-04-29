package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/compute/metadata"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
)

type LevelNameHook struct {
	zerolog.LevelHook
}

func (h LevelNameHook) Run(e *zerolog.Event, l zerolog.Level, msg string) {
	switch l {
	case zerolog.NoLevel:
		e.Str("severity", "DEFAULT")
	case zerolog.DebugLevel:
		e.Str("severity", "DEBUG")
	case zerolog.InfoLevel:
		e.Str("severity", "INFO")
	case zerolog.WarnLevel:
		e.Str("severity", "WARNING")
	case zerolog.ErrorLevel:
		e.Str("severity", "ERROR")
	case zerolog.FatalLevel:
		e.Str("severity", "CRITICAL")
	case zerolog.TraceLevel:
		e.Str("severity", "NOTICE")
	}
}

func LoggerFromContext(ctx context.Context) zerolog.Logger {
	logger := zerolog.Ctx(ctx)
	span := trace.FromContext(ctx).SpanContext()

	return logger.With().
		Str("logging.googleapis.com/spanId", span.SpanID.String()).
		Bool("logging.googleapis.com/trace_sampled", span.IsSampled()).
		Logger()
}

func LogInterceptor(project string, logger *zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		span := trace.FromContext(ctx).SpanContext()

		var trace string
		if metadata.OnGCE() {
			trace = span.TraceID.String()
		} else {
			trace = xid.New().String()
		}
		logger := logger.With().
			Str("logging.googleapis.com/trace", fmt.Sprintf("projects/%s/traces/%s", project, trace)).
			Logger()
		ctx = logger.WithContext(ctx)
		return handler(ctx, req)
	}
}
