package meter

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	MethodAttrKey       attribute.Key = "method"
	ServerStreamAttrKey attribute.Key = "is_server"
	ClientStreamAttrKey attribute.Key = "is_client"
)

func NewStreamMetricInterceptor() grpc.StreamServerInterceptor {
	meter := GetMeter()
	streamGauge := metric.Must(meter).NewInt64UpDownCounter("grpc_server_streams")
	responseTime := metric.Must(meter).NewInt64Histogram("grpc_server_stream_response_ms")
	clientResponseTime := metric.Must(meter).NewInt64Histogram("grpc_server_stream_client_response_ms")

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		methodAttr := MethodAttrKey.String(info.FullMethod)
		serverStreamAttr := ServerStreamAttrKey.Bool(info.IsServerStream)
		clientStreamAttr := ClientStreamAttrKey.Bool(info.IsClientStream)

		ctx := ss.Context()
		streamGauge.Add(ctx, 1, methodAttr, serverStreamAttr, clientStreamAttr)
		r := handler(srv, NewServerStreamW(ctx, ss, responseTime, clientResponseTime, methodAttr, serverStreamAttr, clientStreamAttr))
		streamGauge.Add(ctx, -1, methodAttr, serverStreamAttr, clientStreamAttr)

		return r
	}
}

type ServerStreamW struct {
	ctx                context.Context
	ss                 grpc.ServerStream
	responseTime       metric.Int64Histogram
	clientResponseTime metric.Int64Histogram
	labels             []attribute.KeyValue
	start              time.Time
	clientStart        time.Time
}

func NewServerStreamW(
	ctx context.Context,
	ss grpc.ServerStream,
	responseTime metric.Int64Histogram,
	clientResponseTime metric.Int64Histogram,
	labels ...attribute.KeyValue) *ServerStreamW {
	return &ServerStreamW{
		ctx:                ctx,
		ss:                 ss,
		responseTime:       responseTime,
		clientResponseTime: clientResponseTime,
		labels:             labels,
		start:              time.Now(),
		clientStart:        time.Now(),
	}
}

func (s *ServerStreamW) SetHeader(md metadata.MD) error {
	return s.ss.SetHeader(md)
}

func (s *ServerStreamW) SendHeader(md metadata.MD) error {
	return s.ss.SendHeader(md)
}

func (s *ServerStreamW) SetTrailer(md metadata.MD) {
	s.ss.SetTrailer(md)
}

func (s *ServerStreamW) Context() context.Context {
	return s.ss.Context()
}

func (s *ServerStreamW) SendMsg(m interface{}) error {
	s.responseTime.Record(s.ctx, int64(time.Since(s.start)/time.Millisecond), s.labels...)
	err := s.ss.SendMsg(m)
	s.clientStart = time.Now()
	return err
}

func (s *ServerStreamW) RecvMsg(m interface{}) error {
	s.clientResponseTime.Record(s.ctx, int64(time.Since(s.start)/time.Millisecond), s.labels...)
	err := s.ss.RecvMsg(m)
	s.start = time.Now()
	return err
}
