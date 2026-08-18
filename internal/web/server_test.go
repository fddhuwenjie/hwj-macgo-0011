package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

// errorEnvelope 解析统一错误响应体 {"error":{"code":..., "message":...}}。
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// TestMalformedCreateReturnsJSONError 验证非法 JSON 请求返回 400 及统一 JSON 错误体，
// 而非原先的 200（客户端会当成已创建）。
func TestMalformedCreateReturnsJSONError(t *testing.T) {
	srv, cleanup := setupServer(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPost, "/api/experiments", strings.NewReader("{"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected json content-type, got %q", ct)
	}
	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("error body is not valid JSON: %v (body=%q)", err, rec.Body.String())
	}
	if env.Error.Code != "malformed_json" {
		t.Fatalf("expected error code malformed_json, got %q", env.Error.Code)
	}
	if env.Error.Message == "" {
		t.Fatal("expected non-empty error message")
	}
}

// TestGetUnknownExperimentReturns404 验证获取不存在的实验返回 404 及统一 JSON 错误体。
func TestGetUnknownExperimentReturns404(t *testing.T) {
	srv, cleanup := setupServer(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/api/experiments/does-not-exist", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("error body is not valid JSON: %v (body=%q)", err, rec.Body.String())
	}
	if env.Error.Code != "not_found" {
		t.Fatalf("expected error code not_found, got %q", env.Error.Code)
	}
}

// TestCreateSuccessReturnsJSON 验证成功路径仍返回 200 + JSON 实验对象（契约未被破坏）。
func TestCreateSuccessReturnsJSON(t *testing.T) {
	srv, cleanup := setupServer(t)
	defer cleanup()
	body := []byte(`{"name":"t","description":"d","param_def":{},"input":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/experiments", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected json content-type, got %q", ct)
	}
	var exp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &exp); err != nil {
		t.Fatalf("success body is not valid JSON: %v", err)
	}
	if exp["status"] != "draft" {
		t.Fatalf("expected draft status, got %v", exp["status"])
	}
}

var _ = context.Background
