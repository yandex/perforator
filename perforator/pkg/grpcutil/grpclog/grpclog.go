package grpclog

import (
	"context"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

////////////////////////////////////////////////////////////////////////////////

func logDeadline(ctx context.Context) log.Field {
	deadline, ok := ctx.Deadline()
	if ok {
		return log.Time("deadline", deadline)
	} else {
		return log.Nil("deadline")
	}
}

func logUserAgent(ctx context.Context) log.Field {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return log.Nil("useragent")
	}

	return log.Strings("useragent", md.Get("User-Agent"))
}

////////////////////////////////////////////////////////////////////////////////

type LogInterceptor struct {
	log   xlog.Logger
	skip  map[string]bool
	reqno atomic.Uint64
}

func NewLogInterceptor(log xlog.Logger) *LogInterceptor {
	return &LogInterceptor{log: log, skip: make(map[string]bool)}
}

func (l *LogInterceptor) SkipMethods(methods ...string) *LogInterceptor {
	for _, method := range methods {
		l.skip[method] = true
	}
	return l
}

func (l *LogInterceptor) UnaryServer() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		start := time.Now()
		method := info.FullMethod

		if l.skip[method] {
			return handler(ctx, req)
		}

		ctx = wrapContext(ctx, method, l.reqno.Add(1)-1)

		l := l.log.With(
			log.Time("start", start),
			logDeadline(ctx),
			logUserAgent(ctx),
		)

		l.Info(ctx, "Unary call started")

		defer func() {
			status := status.Convert(err)

			l := l.With(
				log.String("grpc.code", status.Code().String()),
				log.String("grpc.message", status.Message()),
				log.Duration("duration", time.Since(start)),
			)

			if err == nil {
				l.Info(ctx, "Unary call completed")
			} else {
				l.Error(ctx, "Unary call failed", log.Error(err))
			}
		}()

		return handler(ctx, req)
	}
}

type reqnoServerStream struct {
	grpc.ServerStream
	reqno  uint64
	method string
}

func (s *reqnoServerStream) Context() context.Context {
	return wrapContext(s.ServerStream.Context(), s.method, s.reqno)
}

func (l *LogInterceptor) StreamServer() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		start := time.Now()

		method := info.FullMethod
		if l.skip[method] {
			return handler(srv, ss)
		}

		ss = &reqnoServerStream{ss, l.reqno.Add(1) - 1, method}
		ctx := ss.Context()

		l := l.log.With(
			log.Time("start", start),
			logDeadline(ctx),
			logUserAgent(ctx),
		)

		l.Info(ctx, "Stream call started")

		defer func() {
			status := status.Convert(err)

			l := l.With(
				log.String("grpc.code", status.Code().String()),
				log.String("grpc.message", status.Message()),
				log.Duration("duration", time.Since(start)),
			)

			if err == nil {
				l.Info(ctx, "Stream call completed")
			} else {
				l.Error(ctx, "Stream call failed", log.Error(err))
			}
		}()

		return handler(srv, ss)
	}
}

func wrapContext(ctx context.Context, method string, reqno uint64) context.Context {
	return xlog.WrapContext(ctx,
		log.String("grpc.method", method),
		log.UInt64("grpc.reqno", reqno),
	)
}
