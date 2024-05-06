package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

type Outcome string

const (
	Success Outcome = "success"
	Error   Outcome = "error"
)

func (O Outcome) String() string {
	return string(O)
}

var (
	once                             sync.Once
	metricsRouter                    *chi.Mux
	httpRequestDurationHistogram     *prometheus.HistogramVec
	eventProcessingDurationHistogram *prometheus.HistogramVec
	unprocessableEntityCounter       *prometheus.CounterVec
	queueOperationFailureCounter     *prometheus.CounterVec
)

// Init initializes the metrics package.
func Init(metricsPort int) {
	once.Do(func() {
		initMetricsRouter(metricsPort)
		registerMetrics()
	})
}

// initMetricsRouter initializes the metrics router.
func initMetricsRouter(metricsPort int) {
	metricsRouter = chi.NewRouter()
	metricsRouter.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	go func() {
		metricsAddr := fmt.Sprintf(":%d", metricsPort)
		err := http.ListenAndServe(metricsAddr, metricsRouter)
		if err != nil {
			log.Fatal().Err(err).Msgf("error starting metrics server on %s", metricsAddr)
		}
	}()
}

// registerMetrics initializes and register the Prometheus metrics.
func registerMetrics() {
	defaultHistogramBucketsSeconds := []float64{0.1, 0.5, 1, 2.5, 5, 10, 30}

	httpRequestDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of http request durations in seconds.",
			Buckets: defaultHistogramBucketsSeconds,
		},
		[]string{"endpoint", "status"},
	)

	eventProcessingDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "event_processing_duration_seconds",
			Help:    "Histogram of event processing durations in seconds.",
			Buckets: defaultHistogramBucketsSeconds,
		},
		[]string{"queuename", "status", "attempts"},
	)

	unprocessableEntityCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "unprocessable_entity_total",
			Help: "Total number of unprocessable entities from the event processing.",
		},
		[]string{"entity"},
	)

	queueOperationFailureCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "queue_operation_failure_total",
			Help: "Total number of failed queue operations per queue name.",
		},
		[]string{"operation", "queuename"},
	)

	prometheus.MustRegister(
		httpRequestDurationHistogram,
		eventProcessingDurationHistogram,
		unprocessableEntityCounter,
		queueOperationFailureCounter,
	)
}

// StartHttpRequestDurationTimer starts a timer to measure http request handling duration.
func StartHttpRequestDurationTimer(endpoint string) func(statusCode int) {
	startTime := time.Now()
	return func(statusCode int) {
		duration := time.Since(startTime).Seconds()
		httpRequestDurationHistogram.WithLabelValues(
			endpoint,
			fmt.Sprintf("%d", statusCode),
		).Observe(duration)
	}
}

func StartEventProcessingDurationTimer(queuename string, attempts int32) func(statusCode int) {
	startTime := time.Now()
	return func(statusCode int) {
		duration := time.Since(startTime).Seconds()
		eventProcessingDurationHistogram.WithLabelValues(
			queuename,
			fmt.Sprintf("%d", statusCode),
			fmt.Sprintf("%d", attempts),
		).Observe(duration)
	}
}

// RecordUnprocessableEntity increments the unprocessable entity counter.
// This is basically the number of items will show up in the unprocessable entity collection
func RecordUnprocessableEntity(entity string) {
	unprocessableEntityCounter.WithLabelValues(entity).Inc()
}

// RecordQueueOperationFailure increments the queue operation failure counter.
func RecordQueueOperationFailure(operation, queuename string) {
	queueOperationFailureCounter.WithLabelValues(operation, queuename).Inc()
}
