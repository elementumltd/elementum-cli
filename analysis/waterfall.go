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
	"os/exec"
	"runtime"
	"text/template"
)

// RenderHTML generates a self-contained HTML waterfall visualization
// and opens it in the user's default browser.
func RenderHTML(a *ConversationAnalysis) error {
	data, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshal analysis: %w", err)
	}

	f, err := os.CreateTemp("", "ei-waterfall-*.html")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpl, err := template.New("waterfall").Parse(waterfallTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	title := "Conversation Analysis"
	if a.Title != nil && *a.Title != "" {
		title = *a.Title
	}

	err = tmpl.Execute(f, map[string]string{
		"Title":    title,
		"DataJSON": string(data),
	})
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	path := f.Name()
	fmt.Printf("Waterfall saved to %s\n", path)

	return openBrowser("file://" + path)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform %s — open %s manually", runtime.GOOS, url)
	}
	return cmd.Start()
}

const waterfallTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} — ei waterfall</title>
<style>
  :root {
    --bg: #0d1117; --surface: #161b22; --border: #30363d;
    --text: #e6edf3; --muted: #8b949e; --accent: #58a6ff;
    --success: #3fb950; --error: #f85149; --warning: #d29922;
    --thinking: #30363d; --tool: #1f6feb; --tool-success: #238636;
    --tool-error: #da3633; --parallel: #8957e5;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
         background: var(--bg); color: var(--text); padding: 24px; line-height: 1.5; }
  h1 { font-size: 20px; margin-bottom: 4px; color: var(--accent); }
  h2 { font-size: 15px; margin: 20px 0 8px; color: var(--muted); text-transform: uppercase; letter-spacing: 0.5px; }
  .meta { color: var(--muted); font-size: 13px; margin-bottom: 20px; }
  .meta span { margin-right: 16px; }
  .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; margin-bottom: 24px; }
  .stat-card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 12px; }
  .stat-card .label { font-size: 11px; color: var(--muted); text-transform: uppercase; }
  .stat-card .value { font-size: 22px; font-weight: 600; margin-top: 2px; }
  .stat-card .value.error { color: var(--error); }
  .waterfall { position: relative; margin: 12px 0; }
  .turn-header { font-size: 13px; font-weight: 600; color: var(--accent); padding: 8px 0 4px;
                  border-top: 1px solid var(--border); margin-top: 8px; }
  .turn-header:first-child { border-top: none; margin-top: 0; }
  .row { display: flex; align-items: center; height: 28px; margin: 2px 0; position: relative; }
  .row-label { width: 220px; flex-shrink: 0; font-size: 12px; overflow: hidden; text-overflow: ellipsis;
               white-space: nowrap; padding-right: 8px; }
  .row-bar-area { flex: 1; position: relative; height: 20px; }
  .bar { position: absolute; height: 20px; border-radius: 3px; min-width: 2px;
         cursor: pointer; transition: opacity 0.15s; }
  .bar:hover { opacity: 0.8; }
  .bar.tool { background: var(--tool-success); }
  .bar.tool.error { background: var(--tool-error); }
  .bar.tool.pending { background: var(--warning); }
  .bar.tool.outlier { box-shadow: 0 0 0 2px var(--warning); }
  .bar.tool.parallel { background: var(--parallel); }
  .bar.thinking { background: var(--thinking); border: 1px solid var(--border); }
  .row-duration { width: 80px; flex-shrink: 0; text-align: right; font-size: 12px;
                  color: var(--muted); font-variant-numeric: tabular-nums; }
  .badge { display: inline-block; font-size: 10px; padding: 1px 5px; border-radius: 3px;
           margin-left: 6px; vertical-align: middle; }
  .badge.outlier { background: var(--warning); color: #000; }
  .badge.error { background: var(--error); color: #fff; }
  .badge.parallel { background: var(--parallel); color: #fff; }
  .tooltip { display: none; position: fixed; background: var(--surface); border: 1px solid var(--border);
             border-radius: 6px; padding: 10px 12px; font-size: 12px; z-index: 100; max-width: 400px;
             box-shadow: 0 4px 12px rgba(0,0,0,0.4); pointer-events: none; }
  .tooltip.show { display: block; }
  .tooltip .tt-name { font-weight: 600; color: var(--accent); margin-bottom: 4px; }
  .tooltip .tt-row { color: var(--muted); }
  .tooltip .tt-row span { color: var(--text); }
  .by-tool { margin-top: 16px; }
  .by-tool table { width: 100%; border-collapse: collapse; font-size: 13px; }
  .by-tool th { text-align: left; color: var(--muted); font-weight: 500; padding: 6px 12px;
                border-bottom: 1px solid var(--border); }
  .by-tool td { padding: 6px 12px; border-bottom: 1px solid var(--border); }
  .turns-table { margin-top: 16px; }
  .turns-table table { width: 100%; border-collapse: collapse; font-size: 13px; }
  .turns-table th { text-align: left; color: var(--muted); font-weight: 500; padding: 6px 12px;
                    border-bottom: 1px solid var(--border); }
  .turns-table td { padding: 6px 12px; border-bottom: 1px solid var(--border); }
  footer { margin-top: 32px; padding-top: 12px; border-top: 1px solid var(--border);
           font-size: 12px; color: var(--muted); }
</style>
</head>
<body>
<h1>{{.Title}}</h1>
<div class="meta" id="meta"></div>

<div class="stats-grid" id="stats"></div>

<h2>Waterfall</h2>
<div class="waterfall" id="waterfall"></div>

<h2>Turns</h2>
<div class="turns-table" id="turns"></div>

<h2>By Tool</h2>
<div class="by-tool" id="byTool"></div>

<div class="tooltip" id="tooltip"></div>

<footer>Generated by <strong>ei conversation --html</strong></footer>

<script>
const data = {{.DataJSON}};

function fmtMs(ms) {
  if (!ms || ms <= 0) return '-';
  if (ms < 1000) return ms + 'ms';
  return (ms / 1000).toFixed(1) + 's';
}

// Meta
const meta = document.getElementById('meta');
meta.innerHTML = [
  '<span>Agent: <strong>' + esc(data.agentName) + '</strong></span>',
  '<span>Messages: ' + data.messageCount + '</span>',
  '<span>Duration: ' + fmtMs(data.timeline.totalDurationMs) + '</span>',
  '<span>Strategy: ' + data.timingStrategy + '</span>',
].join('');

// Stats cards
const s = data.stats;
let thinkTotal = 0;
(data.thinkingPeriods || []).forEach(p => thinkTotal += p.durationMs);
const statsHtml = [
  card('Tool Calls', s.totalToolCalls),
  card('Avg Duration', fmtMs(s.avgDurationMs)),
  card('P95', fmtMs(s.p95DurationMs)),
  card('Max', fmtMs(s.maxDurationMs)),
  card('Errors', s.errorCount, s.errorCount > 0 ? 'error' : ''),
  card('LLM Thinking', fmtMs(thinkTotal)),
].join('');
document.getElementById('stats').innerHTML = statsHtml;

function card(label, value, cls) {
  return '<div class="stat-card"><div class="label">' + label + '</div><div class="value ' + (cls||'') + '">' + value + '</div></div>';
}

// Waterfall
const wf = document.getElementById('waterfall');
const allCalls = data.toolCalls || [];
const allThinking = data.thinkingPeriods || [];
const turns = (data.timeline && data.timeline.turns) || [];
const timelineStart = new Date(data.timelineStart).getTime();
const timelineEnd = new Date(data.timelineEnd).getTime();
const totalMs = timelineEnd - timelineStart || 1;

turns.forEach((turn, ti) => {
  const turnStart = new Date(turn.userMessageAt).getTime();
  let turnEnd = timelineEnd;
  if (ti + 1 < turns.length) turnEnd = new Date(turns[ti+1].userMessageAt).getTime();

  const hdr = document.createElement('div');
  hdr.className = 'turn-header';
  hdr.textContent = 'Turn ' + (ti+1) + ' — ' + fmtMs(turn.endToEndMs);
  wf.appendChild(hdr);

  // Collect items for this turn
  const items = [];
  allThinking.forEach(tp => {
    const t = new Date(tp.startedAt).getTime();
    if (t >= turnStart && t < turnEnd) items.push({type:'thinking', data:tp, start:t});
  });
  allCalls.forEach(tc => {
    if (!tc.startedAt) return;
    const t = new Date(tc.startedAt).getTime();
    if (t >= turnStart && t < turnEnd) items.push({type:'tool', data:tc, start:t});
  });
  items.sort((a,b) => a.start - b.start);

  items.forEach(item => {
    if (item.type === 'thinking') renderThinkRow(item.data);
    else renderToolRow(item.data);
  });
});

function renderToolRow(tc) {
  const row = document.createElement('div');
  row.className = 'row';
  const startMs = tc.startedAt ? new Date(tc.startedAt).getTime() - timelineStart : 0;
  const durMs = tc.durationMs || 0;
  const leftPct = (startMs / totalMs * 100).toFixed(2);
  const widthPct = Math.max(0.3, durMs / totalMs * 100).toFixed(2);

  let cls = 'bar tool';
  if (tc.status === 'error') cls += ' error';
  else if (tc.status === 'pending') cls += ' pending';
  if (tc.isOutlier) cls += ' outlier';
  if (tc.parallelSize > 1) cls += ' parallel';

  let badges = '';
  if (tc.isOutlier) badges += '<span class="badge outlier">outlier</span>';
  if (tc.status === 'error') badges += '<span class="badge error">error</span>';
  if (tc.parallelSize > 1) badges += '<span class="badge parallel">' + tc.parallelSize + 'x</span>';

  row.innerHTML =
    '<div class="row-label">' + esc(tc.resolvedLabel) + badges + '</div>' +
    '<div class="row-bar-area"><div class="' + cls + '" style="left:' + leftPct + '%;width:' + widthPct + '%" ' +
    'data-tc=\'' + esc(JSON.stringify(tc)) + '\'></div></div>' +
    '<div class="row-duration">' + fmtMs(durMs) + '</div>';
  wf.appendChild(row);
}

function renderThinkRow(tp) {
  const row = document.createElement('div');
  row.className = 'row';
  const startMs = new Date(tp.startedAt).getTime() - timelineStart;
  const durMs = tp.durationMs || 0;
  const leftPct = (startMs / totalMs * 100).toFixed(2);
  const widthPct = Math.max(0.3, durMs / totalMs * 100).toFixed(2);

  row.innerHTML =
    '<div class="row-label" style="color:var(--muted)">LLM thinking</div>' +
    '<div class="row-bar-area"><div class="bar thinking" style="left:' + leftPct + '%;width:' + widthPct + '%"></div></div>' +
    '<div class="row-duration">' + fmtMs(durMs) + '</div>';
  wf.appendChild(row);
}

// Tooltip
const tooltip = document.getElementById('tooltip');
document.addEventListener('mouseover', function(e) {
  const bar = e.target.closest('.bar.tool');
  if (!bar) { tooltip.classList.remove('show'); return; }
  try {
    const tc = JSON.parse(bar.dataset.tc);
    let html = '<div class="tt-name">' + esc(tc.resolvedLabel) + '</div>';
    html += '<div class="tt-row">Status: <span>' + tc.status + '</span></div>';
    html += '<div class="tt-row">Duration: <span>' + fmtMs(tc.durationMs) + '</span></div>';
    html += '<div class="tt-row">Strategy: <span>' + tc.strategy + '</span></div>';
    if (tc.errorMessage) html += '<div class="tt-row" style="color:var(--error)">' + esc(tc.errorMessage) + '</div>';
    if (tc.arguments && tc.arguments.length) {
      html += '<div class="tt-row" style="margin-top:4px">Arguments:</div>';
      tc.arguments.forEach(a => {
        let val = typeof a.value === 'string' ? a.value : JSON.stringify(a.value);
        if (val.length > 60) val = val.substring(0, 57) + '...';
        html += '<div class="tt-row">&nbsp;&nbsp;' + esc(a.name) + ': <span>' + esc(val) + '</span></div>';
      });
    }
    tooltip.innerHTML = html;
    tooltip.classList.add('show');
    const rect = bar.getBoundingClientRect();
    tooltip.style.left = Math.min(rect.left, window.innerWidth - 420) + 'px';
    tooltip.style.top = (rect.bottom + 8) + 'px';
  } catch(ex) {}
});
document.addEventListener('mouseout', function(e) {
  if (e.target.closest('.bar.tool')) tooltip.classList.remove('show');
});

// Turns table
const turnsEl = document.getElementById('turns');
if (turns.length) {
  let html = '<table><tr><th>#</th><th>User Message</th><th>TTFR</th><th>E2E</th><th>Tools</th></tr>';
  turns.forEach((t, i) => {
    let msg = t.userContent || '';
    if (msg.length > 60) msg = msg.substring(0, 57) + '...';
    html += '<tr><td>' + (i+1) + '</td><td>' + esc(msg) + '</td><td>' + fmtMs(t.timeToFirstResponseMs) +
            '</td><td>' + fmtMs(t.endToEndMs) + '</td><td>' + t.toolCallCount + '</td></tr>';
  });
  html += '</table>';
  turnsEl.innerHTML = html;
}

// By-tool table
const byToolEl = document.getElementById('byTool');
const bt = s.byToolName || {};
const names = Object.keys(bt).sort();
if (names.length) {
  let html = '<table><tr><th>Tool</th><th>Count</th><th>Avg</th><th>Min</th><th>Max</th></tr>';
  names.forEach(n => {
    const t = bt[n];
    html += '<tr><td>' + esc(n) + '</td><td>' + t.count + '</td><td>' + fmtMs(t.avgMs) +
            '</td><td>' + fmtMs(t.minMs) + '</td><td>' + fmtMs(t.maxMs) + '</td></tr>';
  });
  html += '</table>';
  byToolEl.innerHTML = html;
}

function esc(s) {
  if (!s) return '';
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}
</script>
</body>
</html>`
