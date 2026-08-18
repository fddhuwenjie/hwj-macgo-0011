package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"experiment-trace/internal/application"
	"experiment-trace/internal/domain"
)

// Server HTTP服务器
type Server struct {
	svc *application.Service
}

// NewServer 创建服务器
func NewServer(svc *application.Service) *Server {
	return &Server{svc: svc}
}

// apiErrorBody 统一错误响应体内部结构。
type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// apiErrorPayload 统一错误响应体：{"error":{"code":..., "message":...}}
type apiErrorPayload struct {
	Error apiErrorBody `json:"error"`
}

// writeJSON 以 application/json 写出成功响应。统一所有成功响应的内容类型与编码方式。
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 以统一 JSON 错误体写出错误响应。
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiErrorPayload{Error: apiErrorBody{Code: code, Message: message}})
}

// statusAndCodeFor 将领域/服务错误映射为 HTTP 状态码与机器可读的错误码。
// 客户端可据此区分“请求有问题”“资源不存在”“状态冲突”等，而不是一律当作成功或 500。
func statusAndCodeFor(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, domain.ErrInvalidParameter):
		return http.StatusBadRequest, "invalid_parameter"
	case errors.Is(err, domain.ErrAlreadyExists),
		errors.Is(err, domain.ErrVersionConflict),
		errors.Is(err, domain.ErrIdempotencyConflict),
		errors.Is(err, domain.ErrInvalidStatus),
		errors.Is(err, domain.ErrImmutableViolation),
		errors.Is(err, domain.ErrLeaseExpired),
		errors.Is(err, domain.ErrLeaseGeneration),
		errors.Is(err, domain.ErrLineageCycle),
		errors.Is(err, domain.ErrTagDrift):
		return http.StatusConflict, errorCodeFor(err)
	case errors.Is(err, domain.ErrBudgetExceeded):
		return http.StatusTooManyRequests, "budget_exceeded"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}

// errorCodeFor 返回冲突类错误的机器可读错误码（仅在状态码确定为 409 时调用）。
func errorCodeFor(err error) string {
	switch {
	case errors.Is(err, domain.ErrAlreadyExists):
		return "already_exists"
	case errors.Is(err, domain.ErrVersionConflict):
		return "version_conflict"
	case errors.Is(err, domain.ErrIdempotencyConflict):
		return "idempotency_conflict"
	case errors.Is(err, domain.ErrInvalidStatus):
		return "invalid_status"
	case errors.Is(err, domain.ErrImmutableViolation):
		return "immutable_violation"
	case errors.Is(err, domain.ErrLeaseExpired):
		return "lease_expired"
	case errors.Is(err, domain.ErrLeaseGeneration):
		return "lease_generation_mismatch"
	case errors.Is(err, domain.ErrLineageCycle):
		return "lineage_cycle"
	case errors.Is(err, domain.ErrTagDrift):
		return "tag_drift"
	default:
		return "conflict"
	}
}

// writeServiceError 将服务层返回的错误映射为统一的错误响应。
func writeServiceError(w http.ResponseWriter, err error) {
	status, code := statusAndCodeFor(err)
	writeError(w, status, code, err.Error())
}

// Handler 返回HTTP处理器
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/api/experiments", s.handleExperiments)
	mux.HandleFunc("/api/experiments/", s.handleExperimentByID)
	mux.HandleFunc("/api/experiments/freeze", s.handleFreeze)
	mux.HandleFunc("/api/experiments/publish", s.handlePublish)
	mux.HandleFunc("/api/runplans", s.handleRunPlans)
	mux.HandleFunc("/", s.handleIndex)
	return mux
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, indexHTML)
}

func (s *Server) handleExperiments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			ParamDef    json.RawMessage `json:"param_def"`
			Input       json.RawMessage `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// 请求体不是合法 JSON：必须返回客户端错误（4xx），否则客户端会按 200 当作已创建。
			writeError(w, http.StatusBadRequest, "malformed_json", "request body is not valid JSON: "+err.Error())
			return
		}
		exp, err := s.svc.CreateExperiment(r.Context(), req.Name, req.Description, req.ParamDef, req.Input, nil)
		if err != nil {
			// 创建失败时服务层已回滚，持久化结果为空；这里返回错误状态码与之保持一致。
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, exp)
	case http.MethodGet:
		exps, err := s.svc.ExpRepo().List(r.Context(), domain.ExperimentFilter{})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, exps)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (s *Server) handleExperimentByID(w http.ResponseWriter, r *http.Request) {
	// 简单路径参数解析
	id := r.URL.Path[len("/api/experiments/"):]
	switch r.Method {
	case http.MethodGet:
		exp, err := s.svc.ExpRepo().Get(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, exp)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (s *Server) handleFreeze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req struct {
		ExperimentID string `json:"experiment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_json", "request body is not valid JSON: "+err.Error())
		return
	}
	plan, err := s.svc.FreezeExperiment(r.Context(), req.ExperimentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req struct {
		ExperimentID string `json:"experiment_id"`
		TagName      string `json:"tag_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_json", "request body is not valid JSON: "+err.Error())
		return
	}
	tag, err := s.svc.PublishExperiment(r.Context(), req.ExperimentID, req.TagName)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tag)
}

func (s *Server) handleRunPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	filter := domain.RunPlanFilter{
		Limit:  10,
		Offset: 0,
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Offset = n
		}
	}
	summaries, err := s.svc.QueryRunPlans(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summaries)
}

// expRepo accessor
func (s *Server) ExpRepo() interface {
	List(ctx context.Context, filter domain.ExperimentFilter) ([]*domain.Experiment, error)
	Get(ctx context.Context, id string) (*domain.Experiment, error)
} {
	return s.svc.ExpRepo()
}

const indexHTML = `<!DOCTYPE html>
<html>
<head>
<title>实验溯源引擎</title>
<style>
body { font-family: sans-serif; margin: 20px; }
.section { margin-bottom: 20px; }
button { margin: 5px; }
table { border-collapse: collapse; width: 100%; }
th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
th { background-color: #f2f2f2; }
</style>
</head>
<body>
<h1>实验运行溯源引擎</h1>
<div class="section">
  <h2>创建实验</h2>
  <input id="expName" placeholder="名称">
  <input id="expDesc" placeholder="描述">
  <textarea id="expParam" placeholder='参数JSON,如{"a":1}'></textarea>
  <textarea id="expInput" placeholder='输入JSON'></textarea>
  <button onclick="createExperiment()">创建</button>
</div>
<div class="section">
  <h2>实验列表</h2>
  <table id="expTable"><tr><th>ID</th><th>名称</th><th>状态</th><th>操作</th></tr></table>
</div>
<div class="section">
  <h2>运行计划摘要</h2>
  <table id="planTable"><tr><th>Plan ID</th><th>Experiment ID</th><th>状态</th><th>重试次数</th><th>耗时</th></tr></table>
</div>
<script>
async function api(url, method='GET', body) {
  const opt = {method, headers: {'Content-Type': 'application/json'}};
  if (body) opt.body = JSON.stringify(body);
  const res = await fetch(url, opt);
  // 非 2xx 一律视为失败：解析统一错误体并抛出，避免客户端把失败响应当成已创建/已成功。
  if (!res.ok) {
    let msg = res.status + ' ' + res.statusText;
    try {
      const j = await res.json();
      if (j && j.error && j.error.message) msg = j.error.message;
    } catch (_) {}
    throw new Error(msg);
  }
  return res.json();
}
async function createExperiment() {
  try {
    const name = document.getElementById('expName').value;
    const description = document.getElementById('expDesc').value;
    const param_def = JSON.parse(document.getElementById('expParam').value || '{}');
    const input = JSON.parse(document.getElementById('expInput').value || '{}');
    const exp = await api('/api/experiments', 'POST', {name, description, param_def, input});
    alert('创建成功:' + exp.id);
    loadExperiments();
  } catch (e) {
    alert('创建失败:' + e.message);
  }
}
async function loadExperiments() {
  const exps = await api('/api/experiments');
  const table = document.getElementById('expTable');
  table.innerHTML = '<tr><th>ID</th><th>名称</th><th>状态</th><th>操作</th></tr>';
  for (const e of exps) {
    const row = table.insertRow();
	    row.innerHTML = '<td>' + e.id + '</td><td>' + e.name + '</td><td>' + e.status + '</td>' +
	      '<td><button onclick="freeze(\'' + e.id + '\')">冻结</button>' +
	      '<button onclick="publish(\'' + e.id + '\')">发布</button></td>';
  }
}
async function freeze(id) {
  try {
    await api('/api/experiments/freeze', 'POST', {experiment_id: id});
    loadExperiments();
  } catch (e) {
    alert('冻结失败:' + e.message);
  }
}
async function publish(id) {
  const tag = prompt('输入发布标签');
  if (!tag) return;
  try {
    await api('/api/experiments/publish', 'POST', {experiment_id: id, tag_name: tag});
    loadExperiments();
  } catch (e) {
    alert('发布失败:' + e.message);
  }
}
async function loadPlans() {
  try {
    const plans = await api('/api/runplans');
    const table = document.getElementById('planTable');
    table.innerHTML = '<tr><th>Plan ID</th><th>Experiment ID</th><th>状态</th><th>重试次数</th><th>耗时</th></tr>';
    for (const p of plans) {
      const row = table.insertRow();
	    row.innerHTML = '<td>' + p.run_plan_id + '</td><td>' + p.experiment_id + '</td><td>' +
	      p.status + '</td><td>' + p.retry_count + '</td><td>' + p.duration + '</td>';
    }
  } catch (e) {
    console.error('加载运行计划失败:', e.message);
  }
}
window.onload = () => { loadExperiments(); loadPlans(); setInterval(loadPlans, 3000); };
</script>
</body>
</html>`
