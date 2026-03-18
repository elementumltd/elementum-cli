// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/elementumltd/elementum-cli/analysis"
	"github.com/elementumltd/elementum-cli/internal/client"
)

// ExecutionServer serves the HTML execution waterfall with lazy I/O loading
type ExecutionServer struct {
	client       *client.Client
	aspectID     string
	automationID string
	executionID  string
	analysis     *analysis.ExecutionAnalysis
	server       *http.Server
	port         int
}

// NewExecutionServer creates a new server for lazy I/O loading
func NewExecutionServer(
	c *client.Client,
	aspectID, automationID, executionID string,
	execAnalysis *analysis.ExecutionAnalysis,
) *ExecutionServer {
	return &ExecutionServer{
		client:       c,
		aspectID:     aspectID,
		automationID: automationID,
		executionID:  executionID,
		analysis:     execAnalysis,
	}
}

// Start starts the HTTP server and opens the browser
func (s *ExecutionServer) Start(ctx context.Context) error {
	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}
	s.port = listener.Addr().(*net.TCPAddr).Port

	// Set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/api/actions/", s.handleActionIO)

	s.server = &http.Server{
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d", s.port)
	fmt.Printf("\nServing execution details at %s\n", url)
	fmt.Println("Press Ctrl+C to stop the server")
	fmt.Println()

	// Open browser
	if err := openBrowser(url); err != nil {
		fmt.Printf("Open %s in your browser\n", url)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		fmt.Println("\nShutting down server...")
	case <-ctx.Done():
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	return nil
}

// handleRoot serves the HTML page with embedded execution metadata
func (s *ExecutionServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data, err := json.Marshal(s.analysis)
	if err != nil {
		http.Error(w, "Failed to marshal data", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("execution").Parse(lazyLoadingHTMLTemplate)
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	title := fmt.Sprintf("Execution %s", truncateID(s.analysis.ExecutionID, 12))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = tmpl.Execute(w, map[string]string{
		"Title":    title,
		"DataJSON": string(data),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Template execution error: %v\n", err)
	}
}

// handleActionIO fetches and returns I/O for a specific action
func (s *ExecutionServer) handleActionIO(w http.ResponseWriter, r *http.Request) {
	// Extract action ID from URL: /api/actions/{id}/io
	path := strings.TrimPrefix(r.URL.Path, "/api/actions/")
	path = strings.TrimSuffix(path, "/io")
	actionID := path

	if actionID == "" {
		http.Error(w, "Action ID required", http.StatusBadRequest)
		return
	}

	// Fetch I/O from GraphQL
	ctx := r.Context()
	resp, err := client.GetActionExecutionIO(ctx, s.client.Genqlient(), s.aspectID, s.automationID, s.executionID, actionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch I/O: %v", err), http.StatusInternalServerError)
		return
	}

	aspectPtr := resp.Organization.Aspect
	if aspectPtr == nil {
		http.Error(w, "Aspect not found", http.StatusNotFound)
		return
	}

	aspect := *aspectPtr
	automation := aspect.GetAutomation()
	if automation == nil {
		http.Error(w, "Automation not found", http.StatusNotFound)
		return
	}

	action := automation.Execution.Action
	if action == nil {
		http.Error(w, "Action not found", http.StatusNotFound)
		return
	}

	// Build response
	result := map[string]any{
		"id":      action.Id,
		"inputs":  nil,
		"outputs": nil,
	}

	if action.InputsOutputs != nil {
		if action.InputsOutputs.Inputs != nil {
			result["inputs"] = *action.InputsOutputs.Inputs
		}
		if action.InputsOutputs.Outputs != nil {
			result["outputs"] = *action.InputsOutputs.Outputs
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func truncateID(id string, length int) string {
	if len(id) <= length {
		return id
	}
	return id[:length] + "..."
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmd, args...).Start()
}

const lazyLoadingHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} — ei execution</title>
<style>
  :root {
    --bg: #0d1117; --surface: #161b22; --border: #30363d;
    --text: #e6edf3; --muted: #8b949e; --accent: #58a6ff;
    --success: #3fb950; --error: #f85149; --warning: #d29922;
    --running: #d29922; --queued: #8957e5;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
         background: var(--bg); color: var(--text); padding: 24px; line-height: 1.5; }
  h1 { font-size: 20px; margin-bottom: 4px; color: var(--accent); }
  h2 { font-size: 15px; margin: 24px 0 12px; color: var(--muted); text-transform: uppercase; letter-spacing: 0.5px; }
  .meta { color: var(--muted); font-size: 13px; margin-bottom: 20px; }
  .meta span { margin-right: 16px; }
  .status { display: inline-block; padding: 3px 10px; border-radius: 12px; font-size: 11px; font-weight: 600; text-transform: uppercase; }
  .status.SUCCESS { background: var(--success); color: #fff; }
  .status.FAILURE { background: var(--error); color: #fff; }
  .status.RUNNING { background: var(--running); color: #000; }
  .status.QUEUED { background: var(--queued); color: #fff; }
  .status.CANCELLED { background: var(--muted); color: #fff; }

  .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 12px; margin-bottom: 24px; }
  .stat-card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 12px; }
  .stat-card .label { font-size: 11px; color: var(--muted); text-transform: uppercase; }
  .stat-card .value { font-size: 22px; font-weight: 600; margin-top: 2px; }
  .stat-card .value.error { color: var(--error); }

  .waterfall { position: relative; margin: 12px 0; }
  .action-row { display: flex; align-items: flex-start; margin: 8px 0; padding: 12px; background: var(--surface);
                border: 1px solid var(--border); border-radius: 8px; }
  .action-row.FAILURE { border-color: var(--error); }
  .action-row.RUNNING { border-color: var(--running); }

  .action-index { width: 32px; flex-shrink: 0; font-size: 14px; font-weight: 600; color: var(--muted); }
  .action-main { flex: 1; min-width: 0; }
  .action-header { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
  .action-name { font-weight: 600; font-size: 14px; color: var(--text); }
  .action-type { font-size: 12px; color: var(--muted); background: var(--bg); padding: 2px 6px; border-radius: 4px; }
  .action-status { font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 4px; }
  .action-status.SUCCESS { background: rgba(63,185,80,0.2); color: var(--success); }
  .action-status.FAILURE { background: rgba(248,81,73,0.2); color: var(--error); }
  .action-status.RUNNING { background: rgba(210,153,34,0.2); color: var(--running); }

  .action-bar-area { margin: 8px 0; height: 20px; background: var(--bg); border-radius: 4px; overflow: hidden; position: relative; }
  .action-bar { height: 100%; border-radius: 4px; min-width: 2px; }
  .action-bar.SUCCESS { background: var(--success); }
  .action-bar.FAILURE { background: var(--error); }
  .action-bar.RUNNING { background: var(--running); }

  .action-duration { width: 80px; flex-shrink: 0; text-align: right; font-size: 13px; color: var(--muted);
                     font-variant-numeric: tabular-nums; padding-top: 4px; }

  .action-error { margin-top: 8px; padding: 8px; background: rgba(248,81,73,0.1); border-radius: 4px;
                  font-size: 12px; color: var(--error); }

  .io-section { margin-top: 8px; }
  .io-toggle { font-size: 12px; color: var(--accent); cursor: pointer; user-select: none; border: none; background: none; padding: 4px 0; }
  .io-toggle:hover { text-decoration: underline; }
  .io-toggle:disabled { color: var(--muted); cursor: wait; }
  .io-content { display: none; margin-top: 8px; padding: 8px; background: var(--bg); border-radius: 4px;
                font-family: monospace; font-size: 11px; white-space: pre-wrap; word-break: break-all;
                max-height: 300px; overflow-y: auto; }
  .io-content.show { display: block; }
  .io-label { font-size: 11px; font-weight: 600; color: var(--muted); margin-bottom: 4px; text-transform: uppercase; }
  .io-value { color: var(--text); }

  .summary-section { margin-top: 24px; padding: 16px; background: var(--surface); border-radius: 8px; }
  .summary-row { display: flex; justify-content: space-between; padding: 4px 0; font-size: 13px; }
  .summary-row .label { color: var(--muted); }
  .summary-row .value { color: var(--text); font-weight: 500; }

  footer { margin-top: 32px; padding-top: 12px; border-top: 1px solid var(--border);
           font-size: 12px; color: var(--muted); }
</style>
</head>
<body>
<h1 id="title"></h1>
<div class="meta" id="meta"></div>

<div class="stats-grid" id="stats"></div>

<h2>Actions</h2>
<div class="waterfall" id="waterfall"></div>

<div class="summary-section" id="summary"></div>

<footer>Generated by <strong>ei automation status --html</strong></footer>

<script>
const data = {{.DataJSON}};

// Track loaded I/O
const loadedIO = {};

function fmtMs(ms) {
  if (!ms || ms <= 0) return '-';
  if (ms < 1000) return ms + 'ms';
  if (ms < 60000) return (ms / 1000).toFixed(1) + 's';
  const mins = Math.floor(ms / 60000);
  const secs = Math.floor((ms % 60000) / 1000);
  return secs > 0 ? mins + 'm ' + secs + 's' : mins + 'm';
}

function esc(s) {
  if (!s) return '';
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function formatJSON(obj) {
  if (!obj) return '(empty)';
  try {
    if (typeof obj === 'string') {
      obj = JSON.parse(obj);
    }
    return JSON.stringify(obj, null, 2);
  } catch (e) {
    return String(obj);
  }
}

// Title
document.getElementById('title').textContent = data.automationName;

// Meta
const meta = document.getElementById('meta');
meta.innerHTML = [
  '<span class="status ' + data.status + '">' + esc(data.status) + '</span>',
  '<span>Version: v' + data.version + '</span>',
  '<span>Duration: ' + fmtMs(data.duration) + '</span>',
  '<span>Execution ID: ' + esc(data.executionId.substring(0, 12)) + '...</span>',
].join('');

// Stats
const s = data.summary || {};
const statsHtml = [
  card('Actions', s.totalActions || 0),
  card('Success', s.successCount || 0),
  card('Failure', s.failureCount || 0, s.failureCount > 0 ? 'error' : ''),
  card('Total Time', fmtMs(s.totalDurationMs)),
  card('Avg Time', fmtMs(s.avgDurationMs)),
  card('Max Time', fmtMs(s.maxDurationMs)),
].join('');
document.getElementById('stats').innerHTML = statsHtml;

function card(label, value, cls) {
  return '<div class="stat-card"><div class="label">' + label + '</div><div class="value ' + (cls||'') + '">' + value + '</div></div>';
}

// Waterfall
const wf = document.getElementById('waterfall');
const actions = data.actions || [];
const maxDur = Math.max(...actions.map(a => a.duration || 0), 1);

actions.forEach((action, i) => {
  const row = document.createElement('div');
  row.className = 'action-row ' + action.status;

  const dur = action.duration || 0;
  const widthPct = Math.max(1, (dur / maxDur) * 100);
  const name = action.name || action.type || 'Unknown';

  let errorHtml = '';
  if (action.error) {
    errorHtml = '<div class="action-error">' + esc(action.error) + '</div>';
  }

  // I/O section with lazy loading button
  const ioId = 'io-' + i;
  const btnId = 'btn-' + i;
  const ioHtml = '<div class="io-section">' +
    '<button id="' + btnId + '" class="io-toggle" onclick="loadIO(\'' + action.id + '\', ' + i + ')">▶ Load Inputs/Outputs</button>' +
    '<div id="' + ioId + '" class="io-content"></div>' +
    '</div>';

  row.innerHTML =
    '<div class="action-index">' + (i + 1) + '</div>' +
    '<div class="action-main">' +
      '<div class="action-header">' +
        '<span class="action-name">' + esc(name) + '</span>' +
        '<span class="action-type">' + esc(action.type) + '</span>' +
        '<span class="action-status ' + action.status + '">' + action.status + '</span>' +
      '</div>' +
      '<div class="action-bar-area"><div class="action-bar ' + action.status + '" style="width:' + widthPct + '%"></div></div>' +
      errorHtml +
      ioHtml +
    '</div>' +
    '<div class="action-duration">' + fmtMs(dur) + '</div>';

  wf.appendChild(row);
});

async function loadIO(actionId, index) {
  const btn = document.getElementById('btn-' + index);
  const container = document.getElementById('io-' + index);

  // Check if already loaded
  if (loadedIO[actionId]) {
    container.classList.toggle('show');
    btn.textContent = container.classList.contains('show') ? '▼ Inputs/Outputs' : '▶ Inputs/Outputs';
    return;
  }

  // Show loading state
  btn.textContent = 'Loading...';
  btn.disabled = true;

  try {
    const resp = await fetch('/api/actions/' + actionId + '/io');
    if (!resp.ok) {
      throw new Error('Failed to load I/O: ' + resp.status);
    }
    const data = await resp.json();

    // Build I/O HTML
    let html = '';
    if (data.inputs) {
      html += '<div class="io-label">Inputs</div><div class="io-value">' + esc(formatJSON(data.inputs)) + '</div>';
    }
    if (data.outputs) {
      html += '<div class="io-label" style="margin-top:8px">Outputs</div><div class="io-value">' + esc(formatJSON(data.outputs)) + '</div>';
    }
    if (!html) {
      html = '<div class="io-value">(no inputs/outputs)</div>';
    }

    container.innerHTML = html;
    container.classList.add('show');
    loadedIO[actionId] = true;
    btn.textContent = '▼ Inputs/Outputs';
    btn.disabled = false;
  } catch (e) {
    container.innerHTML = '<div class="io-value" style="color:var(--error)">Error: ' + esc(e.message) + '</div>';
    container.classList.add('show');
    btn.textContent = '⚠ Error loading I/O';
    btn.disabled = false;
  }
}

// Summary
const summary = document.getElementById('summary');
summary.innerHTML =
  '<div class="summary-row"><span class="label">Started</span><span class="value">' + new Date(data.startedAt).toLocaleString() + '</span></div>' +
  (data.completedAt ? '<div class="summary-row"><span class="label">Completed</span><span class="value">' + new Date(data.completedAt).toLocaleString() + '</span></div>' : '') +
  '<div class="summary-row"><span class="label">Duration</span><span class="value">' + fmtMs(data.duration) + '</span></div>' +
  '<div class="summary-row"><span class="label">Version</span><span class="value">v' + data.version + '</span></div>' +
  (data.errors && data.errors.length ? '<div class="summary-row"><span class="label">Errors</span><span class="value" style="color:var(--error)">' + esc(data.errors.join(', ')) + '</span></div>' : '');
</script>
</body>
</html>`
