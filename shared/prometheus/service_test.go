package prometheus

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared"
)

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

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("expected error status but got %v", rr.Code)
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
