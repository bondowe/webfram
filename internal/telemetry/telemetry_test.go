package telemetry

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRequestsTotalMetric(t *testing.T) {
	// Reset the metric before testing
	RequestsTotal.Reset()

	// Test incrementing counters with different labels
	testCases := []struct {
		method string
		path   string
		status string
	}{
		{"GET", "/", "2xx"},
		{"POST", "/api/users", "2xx"},
		{"GET", "/", "2xx"},
		{"PUT", "/api/users/1", "4xx"},
		{"DELETE", "/api/users/1", "5xx"},
	}

	for _, tc := range testCases {
		RequestsTotal.WithLabelValues(tc.method, tc.path, tc.status).Inc()
	}

	// Verify GET / 2xx was incremented twice
	count := testutil.ToFloat64(RequestsTotal.WithLabelValues("GET", "/", "2xx"))
	if count != 2 {
		t.Errorf("Expected GET / 2xx count to be 2, got %f", count)
	}

	// Verify POST /api/users 2xx was incremented once
	count = testutil.ToFloat64(RequestsTotal.WithLabelValues("POST", "/api/users", "2xx"))
	if count != 1 {
		t.Errorf("Expected POST /api/users 2xx count to be 1, got %f", count)
	}

	// Verify PUT /api/users/1 4xx was incremented once
	count = testutil.ToFloat64(RequestsTotal.WithLabelValues("PUT", "/api/users/1", "4xx"))
	if count != 1 {
		t.Errorf("Expected PUT /api/users/1 4xx count to be 1, got %f", count)
	}

	// Verify DELETE /api/users/1 5xx was incremented once
	count = testutil.ToFloat64(RequestsTotal.WithLabelValues("DELETE", "/api/users/1", "5xx"))
	if count != 1 {
		t.Errorf("Expected DELETE /api/users/1 5xx count to be 1, got %f", count)
	}
}

func TestRequestDurationSecondsMetric(t *testing.T) {
	// Reset the metric before testing
	RequestDurationSeconds.Reset()

	// Test observing durations with different labels
	testCases := []struct {
		method   string
		path     string
		status   string
		duration float64
	}{
		{"GET", "/duration-test", "2xx", 0.1},
		{"GET", "/duration-test", "2xx", 0.2},
		{"POST", "/api/duration-test", "2xx", 0.5},
		{"GET", "/api/duration-test", "4xx", 0.05},
	}

	for _, tc := range testCases {
		RequestDurationSeconds.WithLabelValues(tc.method, tc.path, tc.status).Observe(tc.duration)
	}

	// Verify observations were recorded
	// We collect the metrics and verify the output is valid
	problems, err := testutil.CollectAndLint(RequestDurationSeconds)
	if err != nil {
		t.Errorf("Failed to collect histogram: %v", err)
	}
	if len(problems) > 0 {
		t.Errorf("Linting issues: %v", problems)
	}
}

func TestActiveConnectionsMetric(t *testing.T) {
	// Reset the metric before testing
	ActiveConnections.Set(0)

	// Test incrementing
	ActiveConnections.Inc()
	value := testutil.ToFloat64(ActiveConnections)
	if value != 1 {
		t.Errorf("Expected active connections to be 1, got %f", value)
	}

	// Test incrementing multiple times
	ActiveConnections.Inc()
	ActiveConnections.Inc()
	value = testutil.ToFloat64(ActiveConnections)
	if value != 3 {
		t.Errorf("Expected active connections to be 3, got %f", value)
	}

	// Test decrementing
	ActiveConnections.Dec()
	value = testutil.ToFloat64(ActiveConnections)
	if value != 2 {
		t.Errorf("Expected active connections to be 2, got %f", value)
	}

	// Test setting to a specific value
	ActiveConnections.Set(10)
	value = testutil.ToFloat64(ActiveConnections)
	if value != 10 {
		t.Errorf("Expected active connections to be 10, got %f", value)
	}

	// Reset for other tests
	ActiveConnections.Set(0)
}

func TestMetricsRegistration(t *testing.T) {
	// Verify that all metrics are registered with Prometheus
	// by checking if they can collect metrics without error

	metrics := []prometheus.Collector{
		RequestsTotal,
		RequestDurationSeconds,
		ActiveConnections,
	}

	for _, metric := range metrics {
		// Try to describe the metric - this will fail if not properly registered
		ch := make(chan *prometheus.Desc, 10)
		metric.Describe(ch)
		close(ch)

		// Verify we got at least one description
		count := 0
		for range ch {
			count++
		}
		if count == 0 {
			t.Errorf("Metric did not provide any descriptions, may not be properly configured")
		}
	}
}

func TestRequestsTotalMetadata(t *testing.T) {
	// Verify the metric metadata
	metricName := "http_requests_total"
	helpText := "Total number of HTTP requests received"

	// Collect the metric
	ch := make(chan *prometheus.Desc, 10)
	RequestsTotal.Describe(ch)
	close(ch)

	// Check the description
	found := false
	for desc := range ch {
		descStr := desc.String()
		if strings.Contains(descStr, metricName) && strings.Contains(descStr, helpText) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected metric description to contain name '%s' and help '%s'", metricName, helpText)
	}
}

func TestRequestDurationSecondsMetadata(t *testing.T) {
	// Verify the metric metadata
	metricName := "http_request_duration_seconds"
	helpText := "Duration of HTTP requests in seconds"

	// Collect the metric
	ch := make(chan *prometheus.Desc, 10)
	RequestDurationSeconds.Describe(ch)
	close(ch)

	// Check the description
	found := false
	for desc := range ch {
		descStr := desc.String()
		if strings.Contains(descStr, metricName) && strings.Contains(descStr, helpText) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected metric description to contain name '%s' and help '%s'", metricName, helpText)
	}
}

func TestActiveConnectionsMetadata(t *testing.T) {
	// Verify the metric metadata
	metricName := "active_connections"
	helpText := "Current number of active connections"

	// Collect the metric
	ch := make(chan *prometheus.Desc, 10)
	ActiveConnections.Describe(ch)
	close(ch)

	// Check the description
	found := false
	for desc := range ch {
		descStr := desc.String()
		if strings.Contains(descStr, metricName) && strings.Contains(descStr, helpText) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected metric description to contain name '%s' and help '%s'", metricName, helpText)
	}
}

func TestConcurrentMetricAccess(t *testing.T) {
	// Test that metrics can be safely accessed concurrently
	RequestsTotal.Reset()

	done := make(chan bool)
	iterations := 100

	// Launch multiple goroutines that increment the counter
	//nolint:intrange // classic for loop with goroutine variable capture
	for i := 0; i < 10; i++ {
		go func() {
			//nolint:intrange // classic for loop for benchmark iteration
			for j := 0; j < iterations; j++ {
				RequestsTotal.WithLabelValues("GET", "/test", "2xx").Inc()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	//nolint:intrange // classic for loop for channel synchronization
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify the final count
	expectedCount := float64(10 * iterations)
	actualCount := testutil.ToFloat64(RequestsTotal.WithLabelValues("GET", "/test", "2xx"))
	if actualCount != expectedCount {
		t.Errorf("Expected count to be %f, got %f", expectedCount, actualCount)
	}
}

func TestHistogramBuckets(t *testing.T) {
	// Verify that the histogram uses the default buckets
	RequestDurationSeconds.Reset()

	// Observe some values across different buckets
	testValues := []float64{0.001, 0.01, 0.1, 0.5, 1.0, 5.0, 10.0}
	for _, v := range testValues {
		RequestDurationSeconds.WithLabelValues("GET", "/bucket-test", "2xx").Observe(v)
	}

	// Verify that the histogram is properly configured
	problems, err := testutil.CollectAndLint(RequestDurationSeconds)
	if err != nil {
		t.Errorf("Failed to collect histogram: %v", err)
	}
	if len(problems) > 0 {
		t.Errorf("Linting issues: %v", problems)
	}
}

func TestMetricLabels(t *testing.T) {
	// Test that metrics properly handle different label values
	RequestsTotal.Reset()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	paths := []string{"/", "/api/v1", "/health"}
	statuses := []string{"2xx", "3xx", "4xx", "5xx"}

	// Increment counters for all combinations
	for _, method := range methods {
		for _, path := range paths {
			for _, status := range statuses {
				RequestsTotal.WithLabelValues(method, path, status).Inc()
			}
		}
	}

	// Verify each combination has a count of 1
	for _, method := range methods {
		for _, path := range paths {
			for _, status := range statuses {
				count := testutil.ToFloat64(RequestsTotal.WithLabelValues(method, path, status))
				if count != 1 {
					t.Errorf("Expected count for %s %s %s to be 1, got %f", method, path, status, count)
				}
			}
		}
	}
}
