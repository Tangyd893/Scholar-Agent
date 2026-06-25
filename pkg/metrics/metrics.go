// Package metrics 定义 Prometheus 指标，
// 由 agent-core、tool-service、gateway 共用。
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Agent 指标
	AgentToolCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agent_tool_calls_total",
			Help: "Total number of tool calls made by the agent.",
		},
		[]string{"tool_name", "status"},
	)

	AgentStepsPerQuery = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "agent_steps_per_query",
			Help:    "Number of ReAct steps per query.",
			Buckets: []float64{1, 2, 3, 5, 8, 13},
		},
	)

	LLMRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_request_duration_seconds",
			Help:    "LLM request duration in seconds.",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		},
		[]string{"provider", "model"},
	)

	// Tool Service 指标
	ToolCallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tool_calls_total",
			Help: "Total number of tool executions.",
		},
		[]string{"tool_name", "status"},
	)

	ToolCallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tool_call_duration_seconds",
			Help:    "Tool execution duration in seconds.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10},
		},
		[]string{"tool_name"},
	)

	// Gateway 指标
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(
		AgentToolCalls, AgentStepsPerQuery, LLMRequestDuration,
		ToolCallsTotal, ToolCallDuration,
		HTTPRequestsTotal, HTTPRequestDuration,
	)
}

// Handler 返回 /metrics 端点的 HTTP Handler。
func Handler() http.Handler {
	return promhttp.Handler()
}
