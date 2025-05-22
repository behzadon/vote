package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	ActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_progress",
			Help: "Number of HTTP requests currently being processed",
		},
		[]string{"method", "path"},
	)

	VoteOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vote_operations_total",
			Help: "Total number of voting operations",
		},
		[]string{"operation", "status"},
	)

	PollOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "poll_operations_total",
			Help: "Total number of poll operations",
		},
		[]string{"operation", "status"},
	)

	UserOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_operations_total",
			Help: "Total number of user operations",
		},
		[]string{"operation", "status"},
	)

	CacheOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "status"},
	)
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		method := c.Request.Method
		start := time.Now()

		ActiveRequests.WithLabelValues(method, path).Inc()
		defer ActiveRequests.WithLabelValues(method, path).Dec()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		RequestDuration.WithLabelValues(method, path, status).Observe(duration)
		RequestTotal.WithLabelValues(method, path, status).Inc()

		switch {
		case path == "/api/polls/:id/vote":
			VoteOperations.WithLabelValues("vote", status).Inc()
		case path == "/api/polls/:id/skip":
			VoteOperations.WithLabelValues("skip", status).Inc()
		case path == "/api/polls":
			if method == "POST" {
				PollOperations.WithLabelValues("create", status).Inc()
			} else if method == "GET" {
				PollOperations.WithLabelValues("list", status).Inc()
			}
		case path == "/api/polls/:id":
			PollOperations.WithLabelValues("get", status).Inc()
		case path == "/api/auth/register":
			UserOperations.WithLabelValues("register", status).Inc()
		case path == "/api/auth/login":
			UserOperations.WithLabelValues("login", status).Inc()
		}
	}
}

func RecordCacheOperation(operation string, hit bool) {
	status := "miss"
	if hit {
		status = "hit"
	}
	CacheOperations.WithLabelValues(operation, status).Inc()
}
