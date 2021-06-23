package logger

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/rs/zerolog"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
)

type LevelNameHook struct {
	zerolog.LevelHook
}

func NewLogger() zerolog.Logger {
	if metadata.OnGCE() {
		return zerolog.New(os.Stderr).Hook(LevelNameHook{})
	}
	return zerolog.New(zerolog.ConsoleWriter{
		Out: os.Stderr,
		FormatTimestamp: func(_ interface{}) string {
			return time.Now().Format(time.Stamp)
		},
	})
}

func (h LevelNameHook) Run(e *zerolog.Event, l zerolog.Level, _ string) {
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
	if !metadata.OnGCE() {
		return *logger
	}
	span := trace.FromContext(ctx).SpanContext()

	return logger.With().
		Str("logging.googleapis.com/spanId", span.SpanID.String()).
		Bool("logging.googleapis.com/trace_sampled", span.IsSampled()).
		Logger()
}

func LogInterceptor(project string, logger zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		span := trace.FromContext(ctx).SpanContext()
		l := logger
		if metadata.OnGCE() {
			trace := span.TraceID.String()
			l = logger.With().
				Str("logging.googleapis.com/trace", fmt.Sprintf("projects/%s/traces/%s", project, trace)).
				Logger()
		}
		ctx = l.WithContext(ctx)
		return handler(ctx, req)
	}
}

func LogMiddleware(project string, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			span := trace.FromContext(ctx).SpanContext()
			l := logger
			if metadata.OnGCE() {
				trace := span.TraceID.String()
				l = logger.With().
					Str("logging.googleapis.com/trace", fmt.Sprintf("projects/%s/traces/%s", project, trace)).
					Logger()
			}
			r = r.WithContext(l.WithContext(ctx))
			h.ServeHTTP(w, r)
		})
	}
}
