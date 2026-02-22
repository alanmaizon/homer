package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestPrometheusTextContainsProviderAndConnectorSeries(t *testing.T) {
	ResetForTests()

	RecordProviderCall("mock", "summarize", "success", "none", 25*time.Millisecond)
	RecordProviderCall("mock", "summarize", "error", "timeout", 10*time.Millisecond)
	RecordConnectorCall("google_docs", "import", "error", "connector_forbidden", 5*time.Millisecond)
	RecordConnectorCall("google_docs", "export", "success", "none", 20*time.Millisecond)

	output := PrometheusText()

	expectedSubstrings := []string{
		"# HELP homer_provider_requests_total",
		"homer_provider_requests_total{provider=\"mock\",operation=\"summarize\",status=\"success\",error_category=\"none\"} 1",
		"homer_provider_requests_total{provider=\"mock\",operation=\"summarize\",status=\"error\",error_category=\"timeout\"} 1",
		"# HELP homer_provider_request_duration_seconds",
		"# HELP homer_connector_requests_total",
		"homer_connector_requests_total{connector=\"google_docs\",operation=\"import\",status=\"error\",error_code=\"connector_forbidden\"} 1",
		"homer_connector_requests_total{connector=\"google_docs\",operation=\"export\",status=\"success\",error_code=\"none\"} 1",
		"# HELP homer_connector_request_duration_seconds",
	}

	for _, substring := range expectedSubstrings {
		if !strings.Contains(output, substring) {
			t.Fatalf("expected metrics output to contain %q\noutput:\n%s", substring, output)
		}
	}
}
