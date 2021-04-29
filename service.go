package main

import (
	"context"
	"errors"

	"github.com/komem3/checktrace/protogen"
	"go.opencensus.io/trace"
)

type traceService struct {
	protogen.UnimplementedTraceServiceServer
}

func (t *traceService) Echo(ctx context.Context, msg *protogen.StringMessage) (*protogen.StringMessage, error) {
	ctx, span := trace.StartSpan(ctx, "Echo")
	defer span.End()
	l := LoggerFromContext(ctx)

	l.Info().Str("msg", msg.GetValue()).Msg("get message")

	if msg.GetValue() == "error" {
		l.Error().Msg(msg.GetValue())
		return nil, errors.New(msg.GetValue())
	}

	returnValue := "hello world"

	l.Info().Str("return", returnValue).Msg("return")

	return &protogen.StringMessage{
		Value: returnValue,
	}, nil
}
