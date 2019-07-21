package prometheus

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestLifecycle(t *testing.T) {
	hook := logTest.NewGlobal()
	prometheusService := NewPrometheusService(":2112", nil)

	prometheusService.Start()

	testutil.AssertLogsContain(t, hook, "Starting service")

	prometheusService.Stop()
	testutil.AssertLogsContain(t, hook, "Stopping service")
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
