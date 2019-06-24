package intercepter

import (
	"context"
	"io"
	"time"

	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc/grpclog"

	"github.com/opentracing/opentracing-go/ext"

	"google.golang.org/grpc"

	"github.com/uber/jaeger-lib/metrics"

	"github.com/opentracing/opentracing-go"
	log "github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
)

// MDCarrier custome carrier
type MDCarrier struct {
	metadata.MD
}

// ForeachKey conforms to the TextMapReader interface.
// 这里必须要实现这个 TextMapReader 这个接口
// TextMapReader is the Extract() carrier for the TextMap builtin format. With it,
// the caller can decode a propagated SpanContext as entries in a map of
// unicode strings.
//type TextMapReader interface {
//	// ForeachKey returns TextMap contents via repeated calls to the `handler`
//	// function. If any call to `handler` returns a non-nil error, ForeachKey
//	// terminates and returns that error.
//	//
//	// NOTE: The backing store for the TextMapReader may contain data unrelated
//	// to SpanContext. As such, Inject() and Extract() implementations that
//	// call the TextMapWriter and TextMapReader interfaces must agree on a
//	// prefix or other convention to distinguish their own key:value pairs.
//	//
//	// The "foreach" callback pattern reduces unnecessary copying in some cases
//	// and also allows implementations to hold locks while the map is read.
//	ForeachKey(handler func(key, val string) error) error
//}
func (m MDCarrier) ForeachKey(handler func(key, val string) error) error {
	for k, strs := range m.MD {
		for _, v := range strs {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Set implements Set() of opentracing.TextMapWriter
// 这里也必须要实现
// TextMapWriter is the Inject() carrier for the TextMap builtin format. With
// it, the caller can encode a SpanContext for propagation as entries in a map
// of unicode strings.
//type TextMapWriter interface {
//	// Set a key:value pair to the carrier. Multiple calls to Set() for the
//	// same key leads to undefined behavior.
//	//
//	// NOTE: The backing store for the TextMapWriter may contain data unrelated
//	// to SpanContext. As such, Inject() and Extract() implementations that
//	// call the TextMapWriter and TextMapReader interfaces must agree on a
//	// prefix or other convention to distinguish their own key:value pairs.
//	Set(key, val string)
//}
func (m MDCarrier) Set(key, val string) {
	m.MD[key] = append(m.MD[key], val)
}

// NewJaegerTracer NewJaegerTracer for current service
func NewJaegerTracer(serviceName string, jagentHost string) (tracer opentracing.Tracer, closer io.Closer, err error) {
	cfg := jaegercfg.Configuration{
		ServiceName: serviceName,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  jagentHost,
		},
	}
	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	tracer, closer, err = cfg.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory))

	if err != nil {
		grpclog.Errorf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	return
}

// ClientInterceptor 客户端拦截器
// https://godoc.org/google.golang.org/grpc#UnaryClientInterceptor
func ClientInterceptor(tracer opentracing.Tracer) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, request, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		//一个RPC调用的服务端的span，和RPC服务客户端的span构成ChildOf关系
		var parentCtx opentracing.SpanContext
		parentSpan := opentracing.SpanFromContext(ctx)
		if parentSpan != nil {
			parentCtx = parentSpan.Context()
		}
		span := tracer.StartSpan(
			method,
			opentracing.ChildOf(parentCtx),
			opentracing.Tag{Key: string(ext.Component), Value: "gRPC"},
			ext.SpanKindRPCClient,
		)

		defer span.Finish()
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		err := tracer.Inject(
			span.Context(),
			opentracing.TextMap,
			MDCarrier{md}, // 自定义 carrier
		)

		if err != nil {
			log.Errorf("inject span error :%v", err.Error())
		}

		newCtx := metadata.NewOutgoingContext(ctx, md)
		err = invoker(newCtx, method, request, reply, cc, opts...)

		if err != nil {
			log.Errorf("call error : %v", err.Error())
		}
		return err
	}
}

// ServerInterceptor Server 端的拦截器
func ServerInterceptor(tracer opentracing.Tracer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		spanContext, err := tracer.Extract(
			opentracing.TextMap,
			MDCarrier{md},
		)

		if err != nil && err != opentracing.ErrSpanContextNotFound {
			grpclog.Errorf("extract from metadata err: %v", err)
		} else {
			span := tracer.StartSpan(
				info.FullMethod,
				ext.RPCServerOption(spanContext),
				opentracing.Tag{Key: string(ext.Component), Value: "gRPC"},
				ext.SpanKindRPCServer,
			)
			defer span.Finish()

			ctx = opentracing.ContextWithSpan(ctx, span)
		}

		return handler(ctx, req)

	}

}
