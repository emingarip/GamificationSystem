package metrics

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	// HTTP metrics
	RequestDuration   *prometheus.HistogramVec
	RequestsTotal     *prometheus.CounterVec
	ActiveConnections prometheus.Gauge

	// Rule engine metrics
	RulesEvaluated   *prometheus.CounterVec
	RulesMatched     *prometheus.CounterVec
	RuleEvalDuration *prometheus.HistogramVec
	ActionsExecuted  *prometheus.CounterVec

	// Badge metrics
	BadgesGranted *prometheus.CounterVec

	// WebSocket metrics
	WebSocketConnections   prometheus.Gauge
	WebSocketMessagesTotal *prometheus.CounterVec
	WebSocketMessagesBytes *prometheus.CounterVec

	// Kafka metrics
	KafkaMessagesProcessed *prometheus.CounterVec
	KafkaMessageDuration   *prometheus.HistogramVec
	KafkaProcessingErrors  *prometheus.CounterVec

	// Internal state
	connectionCounter int64
}

// Global metrics instance
var globalMetrics *Metrics
var metricsOnce sync.Once

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	metricsOnce.Do(func() {
		globalMetrics = &Metrics{}
		globalMetrics.registerMetrics()
	})
	return globalMetrics
}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	if globalMetrics == nil {
		return NewMetrics()
	}
	return globalMetrics
}

// registerMetrics registers all Prometheus metrics
func (m *Metrics) registerMetrics() {
	// HTTP metrics
	m.RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"method", "path", "status"},
	)

	m.RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total number of requests",
		},
		[]string{"method", "path", "status"},
	)

	m.ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	// Rule engine metrics
	m.RulesEvaluated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rules_evaluated_total",
			Help: "Total number of rules evaluated",
		},
		[]string{"event_type", "rule_id"},
	)

	m.RulesMatched = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rules_matched_total",
			Help: "Total number of rules that matched",
		},
		[]string{"event_type", "rule_id"},
	)

	m.RuleEvalDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rule_eval_duration_ms",
			Help:    "Rule evaluation duration in milliseconds",
			Buckets: []float64{0.1, 0.5, 1, 5, 10, 25, 50, 100, 250, 500},
		},
		[]string{"event_type", "rule_id"},
	)

	m.ActionsExecuted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "actions_executed_total",
			Help: "Total number of actions executed",
		},
		[]string{"action_type", "rule_id"},
	)

	// Badge metrics
	m.BadgesGranted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "badges_granted_total",
			Help: "Total number of badges granted",
		},
		[]string{"badge_id", "badge_name", "user_id"},
	)

	// WebSocket metrics
	m.WebSocketConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_connections",
			Help: "Number of active WebSocket connections",
		},
	)

	m.WebSocketMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_messages_total",
			Help: "Total number of WebSocket messages",
		},
		[]string{"direction", "message_type"},
	)

	m.WebSocketMessagesBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_messages_bytes_total",
			Help: "Total bytes transferred via WebSocket",
		},
		[]string{"direction"},
	)

	// Kafka metrics
	m.KafkaMessagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_processed_total",
			Help: "Total number of Kafka messages processed",
		},
		[]string{"topic", "partition"},
	)

	m.KafkaMessageDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_message_duration_ms",
			Help:    "Kafka message processing duration in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"topic", "partition"},
	)

	m.KafkaProcessingErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_processing_errors_total",
			Help: "Total number of Kafka processing errors",
		},
		[]string{"topic", "partition", "error_type"},
	)
}

// HTTP middleware helper functions

// RecordRequest records an HTTP request
func (m *Metrics) RecordRequest(method, path string, statusCode int, duration time.Duration) {
	m.RequestDuration.WithLabelValues(method, path, http.StatusText(statusCode)).Observe(duration.Seconds())
	m.RequestsTotal.WithLabelValues(method, path, http.StatusText(statusCode)).Inc()
}

// IncActiveConnections increments the active connections counter
func (m *Metrics) IncActiveConnections() {
	m.ActiveConnections.Inc()
	atomic.AddInt64(&m.connectionCounter, 1)
}

// DecActiveConnections decrements the active connections counter
func (m *Metrics) DecActiveConnections() {
	m.ActiveConnections.Dec()
	atomic.AddInt64(&m.connectionCounter, -1)
}

// GetActiveConnections returns the current number of active connections
func (m *Metrics) GetActiveConnections() int64 {
	return atomic.LoadInt64(&m.connectionCounter)
}

// Rule engine helper functions

// RecordRuleEvaluated records a rule evaluation
func (m *Metrics) RecordRuleEvaluated(eventType, ruleID string) {
	m.RulesEvaluated.WithLabelValues(eventType, ruleID).Inc()
}

// RecordRuleMatched records a matched rule
func (m *Metrics) RecordRuleMatched(eventType, ruleID string) {
	m.RulesMatched.WithLabelValues(eventType, ruleID).Inc()
}

// RecordRuleEvalDuration records rule evaluation duration
func (m *Metrics) RecordRuleEvalDuration(eventType, ruleID string, durationMs float64) {
	m.RuleEvalDuration.WithLabelValues(eventType, ruleID).Observe(durationMs)
}

// RecordActionExecuted records an action execution
func (m *Metrics) RecordActionExecuted(actionType, ruleID string) {
	m.ActionsExecuted.WithLabelValues(actionType, ruleID).Inc()
}

// Badge helper functions

// RecordBadgeGranted records a badge grant
func (m *Metrics) RecordBadgeGranted(badgeID, badgeName, userID string) {
	m.BadgesGranted.WithLabelValues(badgeID, badgeName, userID).Inc()
}

// WebSocket helper functions

// IncWebSocketConnections increments WebSocket connections
func (m *Metrics) IncWebSocketConnections() {
	m.WebSocketConnections.Inc()
}

// DecWebSocketConnections decrements WebSocket connections
func (m *Metrics) DecWebSocketConnections() {
	m.WebSocketConnections.Dec()
}

// RecordWebSocketMessage records a WebSocket message
func (m *Metrics) RecordWebSocketMessage(direction, messageType string, bytes int) {
	m.WebSocketMessagesTotal.WithLabelValues(direction, messageType).Inc()
	m.WebSocketMessagesBytes.WithLabelValues(direction).Add(float64(bytes))
}

// Kafka helper functions

// RecordKafkaMessageProcessed records a processed Kafka message
func (m *Metrics) RecordKafkaMessageProcessed(topic, partition string, offset int64, duration time.Duration) {
	m.KafkaMessagesProcessed.WithLabelValues(topic, partition).Inc()
	m.KafkaMessageDuration.WithLabelValues(topic, partition).Observe(float64(duration.Milliseconds()))
}

// RecordKafkaError records a Kafka processing error
func (m *Metrics) RecordKafkaError(topic, partition, errorType string) {
	m.KafkaProcessingErrors.WithLabelValues(topic, partition, errorType).Inc()
}

// Handler returns the Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// InitializePrometheus initializes the metrics and returns the handler
func InitializePrometheus() http.Handler {
	NewMetrics()
	return promhttp.Handler()
}
