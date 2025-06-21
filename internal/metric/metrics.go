package metric

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"time"
)

const (
	namespace = "bike_domain"
	appName   = "bike_service"
)

type ServerMetrics struct {
	requestsReceived   prometheus.Counter
	responsesSent      *prometheus.CounterVec
	processingDuration *prometheus.HistogramVec
}

var serverMetrics *ServerMetrics

func Init(_ context.Context) error {
	labels := []string{"method", "status"}

	serverMetrics = &ServerMetrics{
		requestsReceived: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "grpc_server",
			Name:      "requests_received_total",
			Help:      "Total number of gRPC requests received",
		}),

		responsesSent: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "grpc_server",
			Name:      "responses_sent_total",
			Help:      "Total number of gRPC responses sent",
		}, labels),

		processingDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "grpc_server",
			Name:      "processing_duration_seconds",
			Help:      "gRPC request processing time distribution",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 16),
		}, labels),
	}

	return nil
}

func MetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		serverMetrics.requestsReceived.Inc()
		start := time.Now()

		res, err := handler(ctx, req)
		duration := time.Since(start)

		grpcStatus, _ := status.FromError(err)
		statusCode := grpcStatus.Code().String()

		methodName := info.FullMethod

		serverMetrics.responsesSent.WithLabelValues(methodName, statusCode).Inc()
		serverMetrics.processingDuration.WithLabelValues(methodName, statusCode).Observe(duration.Seconds())

		return res, err
	}
}
