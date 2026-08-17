package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"experiment-trace/internal/application"
	"experiment-trace/internal/repository"
)

func setupServer(t *testing.T) (*Server, func()) {
	dir, _ := os.MkdirTemp("", "web-*")
	store, _ := repository.NewFileStore(dir)
	expRepo := repository.NewExperimentFileRepo(store)
	paramRepo := repository.NewParamVersionFileRepo(store)
	inputRepo := repository.NewInputSnapshotFileRepo(store)
	planRepo := repository.NewRunPlanFileRepo(store)
	attemptRepo := repository.NewExecutionAttemptFileRepo(store)
	leaseRepo := repository.NewWorkerLeaseFileRepo(store)
	artifactRepo := repository.NewOutputArtifactFileRepo(store)
	lineageRepo := repository.NewLineageEdgeFileRepo(store)
	tagRepo := repository.NewReleaseTagFileRepo(store)
	budgetRepo := repository.NewResourceBudgetFileRepo(store)
	idemRepo := repository.NewIdempotencyFileRepo(store)
	svc := application.NewService(expRepo, paramRepo, inputRepo, planRepo, attemptRepo, leaseRepo, artifactRepo, lineageRepo, tagRepo, budgetRepo, idemRepo)
	srv := NewServer(svc)
	return srv, func() { os.RemoveAll(dir) }
}

func TestHealthz(t *testing.T) {
	srv, cleanup := setupServer(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreateExperimentAPI(t *testing.T) {
	srv, cleanup := setupServer(t)
	defer cleanup()
	body := []byte(`{"name":"test","description":"d","param_def":{},"input":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/experiments", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var exp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &exp); err != nil {
		t.Fatal(err)
	}
	if exp["status"] != "draft" {
		t.Fatalf("expected draft status")
	}
}

var _ = context.Background
