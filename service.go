package checktrace

import (
	"context"
	"errors"

	"github.com/komem3/checktrace/internal/logger"
	"github.com/komem3/checktrace/protogen"
	"go.opencensus.io/trace"
)

type TraceService struct {
	protogen.UnimplementedTraceServiceServer
}

func (t *TraceService) Echo(ctx context.Context, msg *protogen.StringMessage) (*protogen.StringMessage, error) {
	ctx, span := trace.StartSpan(ctx, "Echo")
	defer span.End()
	l := logger.LoggerFromContext(ctx)

	l.Info().Str("msg", msg.GetValue()).Msg("get message")

	if msg.GetValue() == "error" {
		l.Error().Msg(msg.GetValue())
		return nil, errors.New(msg.GetValue())
	}

	returnValue := "hello world"
	if msg.GetValue() != "" {
		returnValue = msg.GetValue()
	}

	l.Info().Str("return", returnValue).Msg("return")

	return &protogen.StringMessage{
		Value: "value=" + returnValue,
	}, nil
}
