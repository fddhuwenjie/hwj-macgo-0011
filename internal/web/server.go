package web

import (
	"context"
	"encoding/json"
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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		exp, err := s.svc.CreateExperiment(r.Context(), req.Name, req.Description, req.ParamDef, req.Input, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(exp)
	case http.MethodGet:
		exps, err := s.svc.ExpRepo().List(r.Context(), domain.ExperimentFilter{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(exps)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleExperimentByID(w http.ResponseWriter, r *http.Request) {
	// 简单路径参数解析
	id := r.URL.Path[len("/api/experiments/"):]
	switch r.Method {
	case http.MethodGet:
		exp, err := s.svc.ExpRepo().Get(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(exp)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleFreeze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ExperimentID string `json:"experiment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plan, err := s.svc.FreezeExperiment(r.Context(), req.ExperimentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(plan)
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ExperimentID string `json:"experiment_id"`
		TagName      string `json:"tag_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tag, err := s.svc.PublishExperiment(r.Context(), req.ExperimentID, req.TagName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tag)
}

func (s *Server) handleRunPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(summaries)
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
  return res.json();
}
async function createExperiment() {
  const name = document.getElementById('expName').value;
  const description = document.getElementById('expDesc').value;
  const param_def = JSON.parse(document.getElementById('expParam').value || '{}');
  const input = JSON.parse(document.getElementById('expInput').value || '{}');
  const exp = await api('/api/experiments', 'POST', {name, description, param_def, input});
  alert('创建成功:' + exp.id);
  loadExperiments();
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
  await api('/api/experiments/freeze', 'POST', {experiment_id: id});
  loadExperiments();
}
async function publish(id) {
  const tag = prompt('输入发布标签');
  if (tag) {
    await api('/api/experiments/publish', 'POST', {experiment_id: id, tag_name: tag});
    loadExperiments();
  }
}
async function loadPlans() {
  const plans = await api('/api/runplans');
  const table = document.getElementById('planTable');
  table.innerHTML = '<tr><th>Plan ID</th><th>Experiment ID</th><th>状态</th><th>重试次数</th><th>耗时</th></tr>';
  for (const p of plans) {
    const row = table.insertRow();
	    row.innerHTML = '<td>' + p.run_plan_id + '</td><td>' + p.experiment_id + '</td><td>' +
	      p.status + '</td><td>' + p.retry_count + '</td><td>' + p.duration + '</td>';
  }
}
window.onload = () => { loadExperiments(); loadPlans(); setInterval(loadPlans, 3000); };
</script>
</body>
</html>`
