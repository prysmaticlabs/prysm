package prometheus

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestLifecycle(t *testing.T) {
	prometheusService := NewPrometheusService(":2112", nil)
	prometheusService.Start()
	// Give service time to start.
	time.Sleep(time.Second)

	// Query the service to ensure it really started.
	resp, err := http.Get("http://localhost:2112/metrics")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if resp.ContentLength == 0 {
		t.Error("Unexpected content length 0")
	}

	err = prometheusService.Stop()
	if err != nil {
		t.Error(err)
	}
	// Give service time to stop.
	time.Sleep(time.Second)

	// Query the service to ensure it really stopped.
	_, err = http.Get("http://localhost:2112/metrics")
	if err == nil {
		t.Fatal("Service still running after Stop()")
	}
}

type mockService struct {
	status error
}

func (m *mockService) Start() {
}

func (m *mockService) Stop() error {
	return nil
}

func (m *mockService) Status() error {
	return m.status
}

func TestHealthz(t *testing.T) {
	registry := shared.NewServiceRegistry()
	m := &mockService{}
	if err := registry.RegisterService(m); err != nil {
		t.Fatalf("failed to registry service %v", err)
	}
	s := NewPrometheusService("" /*addr*/, registry)

	req, err := http.NewRequest("GET", "/healthz", nil /*reader*/)
	if err != nil {
		t.Fatal(err)
	}

	handler := http.HandlerFunc(s.healthzHandler)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected OK status but got %v", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "*prometheus.mockService: OK") {
		t.Errorf("Expected body to contain mockService status, but got %v", body)
	}

	m.status = errors.New("something really bad has happened")

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("expected OK status but got %v", rr.Code)
	}

	body = rr.Body.String()
	if !strings.Contains(
		body,
		"*prometheus.mockService: ERROR something really bad has happened",
	) {
		t.Errorf("Expected body to contain mockService status, but got %v", body)
	}

}

func TestStatus(t *testing.T) {
	failError := errors.New("failure")
	s := &Service{failStatus: failError}

	if err := s.Status(); err != s.failStatus {
		t.Errorf("Wanted: %v, got: %v", s.failStatus, s.Status())
	}
}

func TestContentNegotiation(t *testing.T) {
	t.Run("/healthz all services are ok", func(t *testing.T) {
		registry := shared.NewServiceRegistry()
		m := &mockService{}
		if err := registry.RegisterService(m); err != nil {
			t.Fatalf("failed to registry service %v", err)
		}
		s := NewPrometheusService("", registry)

		req, err := http.NewRequest("GET", "/healthz", nil /* body */)
		if err != nil {
			t.Fatal(err)
		}

		handler := http.HandlerFunc(s.healthzHandler)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		body := rr.Body.String()
		if !strings.Contains(body, "*prometheus.mockService: OK") {
			t.Errorf("Expected body to contain mockService status, but got %q", body)
		}

		// Request response as JSON.
		req.Header.Add("Accept", "application/json, */*;q=0.5")
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		body = rr.Body.String()
		expectedJSON := "{\"error\":\"\",\"data\":[{\"service\":\"*prometheus.mockService\",\"status\":true,\"error\":\"\"}]}"
		if !strings.Contains(body, expectedJSON) {
			t.Errorf("Unexpected data, want: %q got %q", expectedJSON, body)
		}
	})

	t.Run("/healthz failed service", func(t *testing.T) {
		registry := shared.NewServiceRegistry()
		m := &mockService{}
		m.status = errors.New("something is wrong")
		if err := registry.RegisterService(m); err != nil {
			t.Fatalf("failed to registry service %v", err)
		}
		s := NewPrometheusService("", registry)

		req, err := http.NewRequest("GET", "/healthz", nil /* body */)
		if err != nil {
			t.Fatal(err)
		}

		handler := http.HandlerFunc(s.healthzHandler)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		body := rr.Body.String()
		if !strings.Contains(body, "*prometheus.mockService: ERROR something is wrong") {
			t.Errorf("Expected body to contain mockService status, but got %q", body)
		}

		// Request response as JSON.
		req.Header.Add("Accept", "application/json, */*;q=0.5")
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		body = rr.Body.String()
		expectedJSON := "{\"error\":\"\",\"data\":[{\"service\":\"*prometheus.mockService\",\"status\":false,\"error\":\"something is wrong\"}]}"
		if !strings.Contains(body, expectedJSON) {
			t.Errorf("Unexpected data, want: %q got %q", expectedJSON, body)
		}
		if rr.Code < 500 {
			t.Errorf("Expected a server error response code, but got %d", rr.Code)
		}
	})
}
