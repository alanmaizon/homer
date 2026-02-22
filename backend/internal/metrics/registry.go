package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var defaultDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type providerKey struct {
	Provider      string
	Operation     string
	Status        string
	ErrorCategory string
}

type connectorKey struct {
	Connector string
	Operation string
	Status    string
	ErrorCode string
}

type histogram struct {
	buckets []float64
	counts  []uint64
	count   uint64
	sum     float64
}

func newHistogram(buckets []float64) *histogram {
	cloned := append([]float64(nil), buckets...)
	return &histogram{
		buckets: cloned,
		counts:  make([]uint64, len(cloned)),
	}
}

func (h *histogram) Observe(value float64) {
	h.count++
	h.sum += value
	for i, upper := range h.buckets {
		if value <= upper {
			h.counts[i]++
		}
	}
}

type registry struct {
	mu sync.Mutex

	providerRequests map[providerKey]uint64
	providerLatency  map[providerKey]*histogram

	connectorRequests map[connectorKey]uint64
	connectorLatency  map[connectorKey]*histogram
}

func newRegistry() *registry {
	return &registry{
		providerRequests:  make(map[providerKey]uint64),
		providerLatency:   make(map[providerKey]*histogram),
		connectorRequests: make(map[connectorKey]uint64),
		connectorLatency:  make(map[connectorKey]*histogram),
	}
}

var globalRegistry = newRegistry()

func RecordProviderCall(provider string, operation string, status string, errorCategory string, duration time.Duration) {
	globalRegistry.recordProviderCall(providerKey{
		Provider:      provider,
		Operation:     operation,
		Status:        status,
		ErrorCategory: errorCategory,
	}, duration)
}

func RecordConnectorCall(connector string, operation string, status string, errorCode string, duration time.Duration) {
	globalRegistry.recordConnectorCall(connectorKey{
		Connector: connector,
		Operation: operation,
		Status:    status,
		ErrorCode: errorCode,
	}, duration)
}

func PrometheusText() string {
	return globalRegistry.renderPrometheus()
}

func ResetForTests() {
	globalRegistry = newRegistry()
}

func (r *registry) recordProviderCall(key providerKey, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providerRequests[key]++
	h, ok := r.providerLatency[key]
	if !ok {
		h = newHistogram(defaultDurationBuckets)
		r.providerLatency[key] = h
	}
	h.Observe(duration.Seconds())
}

func (r *registry) recordConnectorCall(key connectorKey, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.connectorRequests[key]++
	h, ok := r.connectorLatency[key]
	if !ok {
		h = newHistogram(defaultDurationBuckets)
		r.connectorLatency[key] = h
	}
	h.Observe(duration.Seconds())
}

func (r *registry) renderPrometheus() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var builder strings.Builder

	builder.WriteString("# HELP homer_provider_requests_total Total provider requests.\n")
	builder.WriteString("# TYPE homer_provider_requests_total counter\n")
	providerReqKeys := make([]providerKey, 0, len(r.providerRequests))
	for key := range r.providerRequests {
		providerReqKeys = append(providerReqKeys, key)
	}
	sort.Slice(providerReqKeys, func(i, j int) bool {
		return providerReqKeys[i].String() < providerReqKeys[j].String()
	})
	for _, key := range providerReqKeys {
		builder.WriteString(fmt.Sprintf(
			"homer_provider_requests_total{provider=%q,operation=%q,status=%q,error_category=%q} %d\n",
			key.Provider, key.Operation, key.Status, key.ErrorCategory, r.providerRequests[key],
		))
	}

	builder.WriteString("# HELP homer_provider_request_duration_seconds Provider request duration in seconds.\n")
	builder.WriteString("# TYPE homer_provider_request_duration_seconds histogram\n")
	providerLatencyKeys := make([]providerKey, 0, len(r.providerLatency))
	for key := range r.providerLatency {
		providerLatencyKeys = append(providerLatencyKeys, key)
	}
	sort.Slice(providerLatencyKeys, func(i, j int) bool {
		return providerLatencyKeys[i].String() < providerLatencyKeys[j].String()
	})
	for _, key := range providerLatencyKeys {
		writeHistogram(
			&builder,
			"homer_provider_request_duration_seconds",
			map[string]string{
				"provider":       key.Provider,
				"operation":      key.Operation,
				"status":         key.Status,
				"error_category": key.ErrorCategory,
			},
			r.providerLatency[key],
		)
	}

	builder.WriteString("# HELP homer_connector_requests_total Total connector import/export requests.\n")
	builder.WriteString("# TYPE homer_connector_requests_total counter\n")
	connectorReqKeys := make([]connectorKey, 0, len(r.connectorRequests))
	for key := range r.connectorRequests {
		connectorReqKeys = append(connectorReqKeys, key)
	}
	sort.Slice(connectorReqKeys, func(i, j int) bool {
		return connectorReqKeys[i].String() < connectorReqKeys[j].String()
	})
	for _, key := range connectorReqKeys {
		builder.WriteString(fmt.Sprintf(
			"homer_connector_requests_total{connector=%q,operation=%q,status=%q,error_code=%q} %d\n",
			key.Connector, key.Operation, key.Status, key.ErrorCode, r.connectorRequests[key],
		))
	}

	builder.WriteString("# HELP homer_connector_request_duration_seconds Connector request duration in seconds.\n")
	builder.WriteString("# TYPE homer_connector_request_duration_seconds histogram\n")
	connectorLatencyKeys := make([]connectorKey, 0, len(r.connectorLatency))
	for key := range r.connectorLatency {
		connectorLatencyKeys = append(connectorLatencyKeys, key)
	}
	sort.Slice(connectorLatencyKeys, func(i, j int) bool {
		return connectorLatencyKeys[i].String() < connectorLatencyKeys[j].String()
	})
	for _, key := range connectorLatencyKeys {
		writeHistogram(
			&builder,
			"homer_connector_request_duration_seconds",
			map[string]string{
				"connector":  key.Connector,
				"operation":  key.Operation,
				"status":     key.Status,
				"error_code": key.ErrorCode,
			},
			r.connectorLatency[key],
		)
	}

	return builder.String()
}

func writeHistogram(builder *strings.Builder, metricName string, labels map[string]string, h *histogram) {
	cumulative := uint64(0)
	for i, bucket := range h.buckets {
		cumulative = h.counts[i]
		builder.WriteString(fmt.Sprintf(
			"%s_bucket{%s,le=%q} %d\n",
			metricName,
			formatLabels(labels),
			formatFloat(bucket),
			cumulative,
		))
	}
	builder.WriteString(fmt.Sprintf(
		"%s_bucket{%s,le=\"+Inf\"} %d\n",
		metricName,
		formatLabels(labels),
		h.count,
	))
	builder.WriteString(fmt.Sprintf("%s_sum{%s} %g\n", metricName, formatLabels(labels), h.sum))
	builder.WriteString(fmt.Sprintf("%s_count{%s} %d\n", metricName, formatLabels(labels), h.count))
}

func formatLabels(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", key, labels[key]))
	}
	return strings.Join(parts, ",")
}

func formatFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", value), "0"), ".")
}

func (k providerKey) String() string {
	return strings.Join([]string{k.Provider, k.Operation, k.Status, k.ErrorCategory}, "|")
}

func (k connectorKey) String() string {
	return strings.Join([]string{k.Connector, k.Operation, k.Status, k.ErrorCode}, "|")
}
