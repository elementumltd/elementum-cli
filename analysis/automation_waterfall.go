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

// RenderAutomationHTML generates a self-contained HTML visualization of the
// automation configuration flow and opens it in the user's default browser.
func RenderAutomationHTML(cfg *AutomationConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal automation config: %w", err)
	}

	f, err := os.CreateTemp("", "ei-automation-*.html")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpl, err := template.New("automation").Parse(automationHTMLTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	err = tmpl.Execute(f, map[string]string{
		"Title":    cfg.AutomationName,
		"DataJSON": string(data),
	})
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	path := f.Name()
	fmt.Printf("Automation config saved to %s\n", path)

	return openBrowser("file://" + path)
}

const automationHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} — Automation Flow</title>
<style>
  :root {
    --bg: #0d1117; --surface: #161b22; --border: #30363d;
    --text: #e6edf3; --muted: #8b949e; --accent: #58a6ff;
    --trigger: #d29922; --task: #238636; --switch: #8957e5;
    --foreach: #58a6ff; --case-bg: #1c1e26; --connector: #484f58;
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
    background: var(--bg); color: var(--text); padding: 32px; line-height: 1.5;
  }
  h1 { font-size: 24px; margin-bottom: 8px; color: var(--text); }
  .meta { color: var(--muted); font-size: 13px; margin-bottom: 32px; }
  .meta span { margin-right: 16px; }
  .status { display: inline-block; padding: 3px 10px; border-radius: 12px; font-size: 11px; font-weight: 600; text-transform: uppercase; }
  .status.active { background: #238636; color: #fff; }
  .status.inactive { background: #da3633; color: #fff; }
  .status.draft { background: #d29922; color: #000; }

  .flow { display: flex; flex-direction: column; align-items: center; gap: 0; margin: 0 auto; max-width: 600px; }
  .connector { width: 2px; height: 20px; background: var(--connector); }

  .node { width: 100%; border-radius: 8px; background: var(--surface); border: 2px solid var(--border); overflow: hidden; }
  .node-header { display: flex; align-items: center; gap: 12px; padding: 14px 18px; }
  .node-icon { font-size: 20px; }
  .node-index { font-size: 13px; color: var(--muted); font-weight: 600; min-width: 28px; }
  .node-content { flex: 1; }
  .node-title { font-weight: 600; font-size: 14px; color: var(--text); }
  .node-type { font-size: 12px; color: var(--muted); margin-top: 2px; }

  .node.trigger { border-color: var(--trigger); }
  .node.trigger .node-icon { color: var(--trigger); }
  .node.task { border-color: var(--task); }
  .node.task .node-icon { color: var(--task); }
  .node.switch { border-color: var(--switch); }
  .node.switch .node-icon { color: var(--switch); }
  .node.foreach { border-color: var(--foreach); }
  .node.foreach .node-icon { color: var(--foreach); }

  .cases { padding: 0 18px 18px; }
  .case-branch { margin-top: 12px; border: 1px solid var(--border); border-radius: 6px; background: var(--case-bg); overflow: hidden; }
  .case-header { padding: 10px 14px; font-size: 12px; font-weight: 600; color: var(--switch); border-bottom: 1px solid var(--border); background: rgba(137,87,229,0.08); }
  .case-body { padding: 14px; display: flex; flex-direction: column; align-items: center; gap: 0; }
  .case-body .node { border-width: 1px; }
  .case-body .connector { height: 14px; }
  .case-empty { color: var(--muted); font-style: italic; font-size: 12px; padding: 8px; text-align: center; }

  .loop-body { margin: 0 18px 18px; border: 2px dashed var(--foreach); border-radius: 8px; padding: 18px; display: flex; flex-direction: column; align-items: center; gap: 0; background: rgba(88,166,255,0.03); }
  .loop-body .node { border-width: 1px; }
  .loop-body .connector { height: 14px; }
  .loop-label { font-size: 11px; color: var(--foreach); text-transform: uppercase; letter-spacing: 1px; margin-bottom: 12px; align-self: flex-start; font-weight: 600; }

  .summary { display: flex; gap: 24px; margin: 32px auto 0; max-width: 600px; justify-content: center; flex-wrap: wrap; }
  .stat { text-align: center; }
  .stat-value { font-size: 28px; font-weight: 700; color: var(--accent); }
  .stat-label { font-size: 11px; color: var(--muted); text-transform: uppercase; letter-spacing: 0.5px; }

  footer { margin-top: 40px; font-size: 12px; color: var(--muted); text-align: center; }
</style>
</head>
<body>
<h1 id="title"></h1>
<div class="meta" id="meta"></div>
<div class="flow" id="flow"></div>
<div class="summary" id="summary"></div>
<footer>Generated by <strong>ei automation config --html</strong></footer>

<script>
const data = {{.DataJSON}};
const flow = document.getElementById('flow');

document.getElementById('title').textContent = data.automationName;

const meta = document.getElementById('meta');
let statusCls = 'draft';
if (data.status === 'ACTIVE') statusCls = 'active';
else if (data.status === 'INACTIVE') statusCls = 'inactive';
meta.innerHTML = '<span class="status ' + statusCls + '">' + esc(data.status) + '</span>';

if (data.trigger) {
  flow.appendChild(createNode(data.trigger, 'trigger', '⚡', ''));
}

(data.tasks || []).forEach(function(task) {
  addConnector();
  renderNode(task);
});

function renderNode(task) {
  if (task.nodeKind === 'switch') {
    renderSwitch(task);
  } else if (task.nodeKind === 'for_each') {
    renderForEach(task);
  } else {
    flow.appendChild(createNode(task, 'task', '●', task.index));
  }
}

function renderSwitch(task) {
  const node = createNode(task, 'switch', '◆', task.index);
  const cases = task.cases || [];
  if (cases.length) {
    const casesDiv = document.createElement('div');
    casesDiv.className = 'cases';
    cases.forEach(function(c) {
      const branch = document.createElement('div');
      branch.className = 'case-branch';
      const hdr = document.createElement('div');
      hdr.className = 'case-header';
      hdr.textContent = c.label || 'Default';
      branch.appendChild(hdr);
      const body = document.createElement('div');
      body.className = 'case-body';
      if (c.tasks && c.tasks.length) {
        c.tasks.forEach(function(ct, ci) {
          if (ci > 0) addConnectorTo(body);
          renderNodeInto(ct, body);
        });
      } else {
        const empty = document.createElement('div');
        empty.className = 'case-empty';
        empty.textContent = '(empty)';
        body.appendChild(empty);
      }
      branch.appendChild(body);
      casesDiv.appendChild(branch);
    });
    node.appendChild(casesDiv);
  }
  flow.appendChild(node);
}

function renderForEach(task) {
  const node = createNode(task, 'foreach', '⟳', task.index);
  const body = task.loopBody || [];
  if (body.length) {
    const loopDiv = document.createElement('div');
    loopDiv.className = 'loop-body';
    const lbl = document.createElement('div');
    lbl.className = 'loop-label';
    lbl.textContent = 'Loop' + (task.loopList ? ': ' + task.loopList : '');
    loopDiv.appendChild(lbl);
    body.forEach(function(bt, bi) {
      if (bi > 0) addConnectorTo(loopDiv);
      renderNodeInto(bt, loopDiv);
    });
    node.appendChild(loopDiv);
  }
  flow.appendChild(node);
}

function renderNodeInto(task, container) {
  if (task.nodeKind === 'switch') {
    const node = createNode(task, 'switch', '◆', task.index);
    const cases = task.cases || [];
    if (cases.length) {
      const casesDiv = document.createElement('div');
      casesDiv.className = 'cases';
      cases.forEach(function(c) {
        const branch = document.createElement('div');
        branch.className = 'case-branch';
        const hdr = document.createElement('div');
        hdr.className = 'case-header';
        hdr.textContent = c.label || 'Default';
        branch.appendChild(hdr);
        const body = document.createElement('div');
        body.className = 'case-body';
        if (c.tasks && c.tasks.length) {
          c.tasks.forEach(function(ct, ci) {
            if (ci > 0) addConnectorTo(body);
            renderNodeInto(ct, body);
          });
        } else {
          const empty = document.createElement('div');
          empty.className = 'case-empty';
          empty.textContent = '(empty)';
          body.appendChild(empty);
        }
        branch.appendChild(body);
        casesDiv.appendChild(branch);
      });
      node.appendChild(casesDiv);
    }
    container.appendChild(node);
  } else if (task.nodeKind === 'for_each') {
    const node = createNode(task, 'foreach', '⟳', task.index);
    const body = task.loopBody || [];
    if (body.length) {
      const loopDiv = document.createElement('div');
      loopDiv.className = 'loop-body';
      const lbl = document.createElement('div');
      lbl.className = 'loop-label';
      lbl.textContent = 'Loop' + (task.loopList ? ': ' + task.loopList : '');
      loopDiv.appendChild(lbl);
      body.forEach(function(bt, bi) {
        if (bi > 0) addConnectorTo(loopDiv);
        renderNodeInto(bt, loopDiv);
      });
      node.appendChild(loopDiv);
    }
    container.appendChild(node);
  } else {
    container.appendChild(createNode(task, 'task', '●', task.index));
  }
}

function createNode(item, cls, icon, index) {
  const node = document.createElement('div');
  node.className = 'node ' + cls;

  const header = document.createElement('div');
  header.className = 'node-header';

  const iconEl = document.createElement('span');
  iconEl.className = 'node-icon';
  iconEl.textContent = icon;
  header.appendChild(iconEl);

  if (index) {
    const idxEl = document.createElement('span');
    idxEl.className = 'node-index';
    idxEl.textContent = index;
    header.appendChild(idxEl);
  }

  const content = document.createElement('div');
  content.className = 'node-content';
  
  const titleEl = document.createElement('div');
  titleEl.className = 'node-title';
  titleEl.textContent = item.name || item.type || 'Unnamed';
  content.appendChild(titleEl);

  const typeEl = document.createElement('div');
  typeEl.className = 'node-type';
  typeEl.textContent = item.type;
  content.appendChild(typeEl);

  header.appendChild(content);
  node.appendChild(header);

  return node;
}

function addConnector() { addConnectorTo(flow); }

function addConnectorTo(parent) {
  const conn = document.createElement('div');
  conn.className = 'connector';
  parent.appendChild(conn);
}

const sm = data.summary || {};
document.getElementById('summary').innerHTML = [
  stat(sm.totalTasks, 'Tasks'),
  sm.switchCount ? stat(sm.switchCount, 'Switches') : '',
  sm.forEachCount ? stat(sm.forEachCount, 'Loops') : '',
].filter(Boolean).join('');

function stat(value, label) {
  return '<div class="stat"><div class="stat-value">' + value + '</div><div class="stat-label">' + label + '</div></div>';
}

function esc(s) {
  if (!s) return '';
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}
</script>
</body>
</html>`
