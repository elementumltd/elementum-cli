// Copyright 2026 Elementum Ltd. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"text/template"
)

// RenderExecutionHTML generates a self-contained HTML waterfall visualization
// of an automation execution with action timing, inputs/outputs, and errors.
// Opens the file in the user's default browser.
func RenderExecutionHTML(exec *ExecutionAnalysis) error {
	data, err := json.Marshal(exec)
	if err != nil {
		return fmt.Errorf("failed to marshal execution analysis: %w", err)
	}

	f, err := os.CreateTemp("", "ei-execution-*.html")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpl, err := template.New("execution").Parse(executionHTMLTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	title := fmt.Sprintf("Execution %s", truncateID(exec.ExecutionID, 12))

	err = tmpl.Execute(f, map[string]string{
		"Title":    title,
		"DataJSON": string(data),
	})
	f.Close()
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	path := f.Name()
	fmt.Printf("Execution analysis saved to %s\n", path)

	return openBrowser("file://" + path)
}

const executionHTMLTemplate = `<!DOCTYPE html>
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
  .io-toggle { font-size: 12px; color: var(--accent); cursor: pointer; user-select: none; }
  .io-toggle:hover { text-decoration: underline; }
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

  let ioHtml = '';
  const hasInputs = action.inputs && typeof action.inputs === 'object' && Object.keys(action.inputs).length > 0;
  const hasOutputs = action.outputs && typeof action.outputs === 'object' && Object.keys(action.outputs).length > 0;
  const hasIO = hasInputs || hasOutputs;

  if (hasIO) {
    const ioId = 'io-' + i;
    ioHtml = '<div class="io-section">' +
      '<span class="io-toggle" onclick="toggleIO(\'' + ioId + '\')">\u25B6 Inputs/Outputs</span>' +
      '<div id="' + ioId + '" class="io-content">';
    if (hasInputs) {
      ioHtml += '<div class="io-label">Inputs</div><div class="io-value">' + esc(formatJSON(action.inputs)) + '</div>';
    }
    if (hasOutputs) {
      ioHtml += '<div class="io-label" style="margin-top:8px">Outputs</div><div class="io-value">' + esc(formatJSON(action.outputs)) + '</div>';
    }
    ioHtml += '</div></div>';
  }

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

function toggleIO(id) {
  const el = document.getElementById(id);
  if (el) el.classList.toggle('show');
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
