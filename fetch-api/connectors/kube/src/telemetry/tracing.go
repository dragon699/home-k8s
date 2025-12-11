package telemetry

import (
	"context"

	"common/telemetry"
	"connector-kube/settings"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
)

var Tracer trace.Tracer
var tracerProvider *telemetry.TracerProvider


func init() {
	var err error

	tracerProvider, err = telemetry.NewTracer(telemetry.TracerConfig{
		ServiceName:      settings.Config.OtelServiceName,
		ServiceNamespace: settings.Config.OtelServiceNamespace,
		ServiceVersion:   settings.Config.OtelServiceVersion,
		OtlpEndpointGrpc: settings.Config.OtlpEndpointGrpc,
	})

	if err != nil {
		Log.Error("Failed to initialize tracer", "error", err.Error())
		panic(err)
	}

	Tracer = tracerProvider.Tracer
}


func ShutdownTracer(ctx context.Context) error {
	if tracerProvider != nil {
		return tracerProvider.Shutdown(ctx)
	}
	return nil
}


func TracingMiddleware() fiber.Handler {
	return otelfiber.Middleware(
		otelfiber.WithSpanNameFormatter(func(ctx *fiber.Ctx) string {
			return ctx.Method() + " " + ctx.Path()
		}),
	)
}


func Traced(ctx context.Context, spanName string, fn func(context.Context, trace.Span) error) error {
	ctx, span := Tracer.Start(ctx, spanName)
	defer span.End()

	return fn(ctx, span)
}


func TracedWithResult[T any](ctx context.Context, spanName string, fn func(context.Context, trace.Span) (T, error)) (T, error) {
	ctx, span := Tracer.Start(ctx, spanName)
	defer span.End()

	return fn(ctx, span)
}


func TracedNoError(ctx context.Context, spanName string, fn func(context.Context, trace.Span)) {
	ctx, span := Tracer.Start(ctx, spanName)
	defer span.End()

	fn(ctx, span)
}
