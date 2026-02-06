package api

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

type uiData struct {
	Lang               string
	Title              string
	Heading            string
	ModuleSectionTitle string
	InstallButton      string
	UninstallButton    string
	CheckButton        string
	RefreshButton      string
	NameLabel          string
	CategoryLabel      string
	VersionLabel       string
	StatusLabel        string
	ActionsLabel       string
	StatusInstalled    string
	StatusNotInstalled string
	StatusUnknown      string
	StatusIdle         string
	StatusLoading      string
	StatusError        string
	StatusReady        string
	EmptyHint          string
	RunningLabel       string
	ErrorPrefix        string
	OKLabel            string
}

func (s *Server) uiHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, i18n.T("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	lang := "en"
	if s.cfg != nil && s.cfg.I18n.Language != "" {
		lang = s.cfg.I18n.Language
	}

	data := uiData{
		Lang:               lang,
		Title:              stripQuotes(i18n.T("LocalAIStack - Module Manager")),
		Heading:            stripQuotes(i18n.T("LocalAIStack Module Manager")),
		ModuleSectionTitle: stripQuotes(i18n.T("Modules")),
		InstallButton:      stripQuotes(i18n.T("Install")),
		UninstallButton:    stripQuotes(i18n.T("Uninstall")),
		CheckButton:        stripQuotes(i18n.T("Check")),
		RefreshButton:      stripQuotes(i18n.T("Refresh")),
		NameLabel:          stripQuotes(i18n.T("Name")),
		CategoryLabel:      stripQuotes(i18n.T("Category")),
		VersionLabel:       stripQuotes(i18n.T("Version")),
		StatusLabel:        stripQuotes(i18n.T("Status")),
		ActionsLabel:       stripQuotes(i18n.T("Actions")),
		StatusInstalled:    stripQuotes(i18n.T("Installed")),
		StatusNotInstalled: stripQuotes(i18n.T("Not installed")),
		StatusUnknown:      stripQuotes(i18n.T("Unknown")),
		StatusIdle:         stripQuotes(i18n.T("Ready")),
		StatusLoading:      stripQuotes(i18n.T("Loading...")),
		StatusError:        stripQuotes(i18n.T("Error")),
		StatusReady:        stripQuotes(i18n.T("Updated")),
		EmptyHint:          stripQuotes(i18n.T("No modules found.")),
		RunningLabel:       stripQuotes(i18n.T("Running...")),
		ErrorPrefix:        stripQuotes(i18n.T("Error")),
		OKLabel:            stripQuotes(i18n.T("OK")),
	}

	tmpl := template.Must(template.New("ui").Parse(uiHTML))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, i18n.T("failed to render page: %v", err), http.StatusInternalServerError)
		return
	}
}

func stripQuotes(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= 2 {
		first := trimmed[0]
		last := trimmed[len(trimmed)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		}
	}
	return trimmed
}

const uiHTML = `<!doctype html>
<html lang="{{.Lang}}">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{.Title}}</title>
  <style>
    :root {
      --bg: #f5f2ed;
      --panel: #ffffff;
      --text: #1f1f1f;
      --muted: #6b6b6b;
      --accent: #1f4d43;
      --accent-2: #d36b2d;
      --border: #e6dfd6;
      --shadow: rgba(0, 0, 0, 0.08);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Space Grotesk", "Saira", "Arial Narrow", sans-serif;
      background: radial-gradient(1200px 800px at 10% 0%, #f7efe3 0%, #f1e8db 45%, #ece4d9 100%);
      color: var(--text);
    }
    .page {
      min-height: 100vh;
      padding: 32px 20px 48px;
      display: flex;
      justify-content: center;
    }
    .container {
      width: min(960px, 100%);
      display: grid;
      gap: 20px;
    }
    header {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    h1 {
      margin: 0;
      font-size: clamp(24px, 4vw, 36px);
      letter-spacing: -0.02em;
    }
    .panel {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 16px;
      padding: 20px;
      box-shadow: 0 12px 30px var(--shadow);
    }
    .row {
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
      align-items: center;
    }
    label {
      font-weight: 600;
    }
    input[type="text"] {
      flex: 1 1 260px;
      min-width: 200px;
      padding: 10px 12px;
      border-radius: 10px;
      border: 1px solid var(--border);
      font-size: 14px;
    }
    .hint {
      color: var(--muted);
      font-size: 13px;
    }
    .table-wrap {
      overflow-x: auto;
      margin-top: 12px;
      border: 1px solid var(--border);
      border-radius: 12px;
      background: #fcfaf7;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 14px;
    }
    thead {
      background: #f1e9dd;
      text-align: left;
    }
    th, td {
      padding: 12px 14px;
      border-bottom: 1px solid var(--border);
      vertical-align: middle;
    }
    tbody tr:hover {
      background: #f7f2ea;
    }
    .actions {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
    }
    button {
      border: none;
      border-radius: 999px;
      padding: 10px 18px;
      font-weight: 600;
      cursor: pointer;
      transition: transform 0.1s ease, box-shadow 0.1s ease;
    }
    button.primary { background: var(--accent); color: #fff; }
    button.secondary { background: var(--accent-2); color: #fff; }
    button.ghost { background: #f3ede4; color: #3e3a35; }
    button:active { transform: translateY(1px); }
    .status {
      font-size: 13px;
      color: var(--muted);
      margin-top: 8px;
    }
    @media (max-width: 720px) {
      .row { flex-direction: column; align-items: stretch; }
      button { width: 100%; }
    }
  </style>
</head>
<body>
  <div class="page">
    <div class="container">
      <header>
        <h1>{{.Heading}}</h1>
        <div class="status">{{.ModuleSectionTitle}}</div>
      </header>

      <section class="panel">
        <div class="row" style="justify-content: space-between;">
          <strong>{{.ModuleSectionTitle}}</strong>
          <div class="row" style="gap: 8px;">
            <button class="ghost" id="refreshButton">{{.RefreshButton}}</button>
          </div>
        </div>
        <div class="status" id="statusText">{{.StatusIdle}}</div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>{{.NameLabel}}</th>
                <th>{{.CategoryLabel}}</th>
                <th>{{.VersionLabel}}</th>
                <th>{{.StatusLabel}}</th>
                <th>{{.ActionsLabel}}</th>
              </tr>
            </thead>
            <tbody id="moduleTableBody">
            </tbody>
          </table>
        </div>
        <div id="emptyHint" class="hint" style="display:none;">{{.EmptyHint}}</div>
      </section>
    </div>
  </div>

  <script>
    const statusText = document.getElementById("statusText");
    const tableBody = document.getElementById("moduleTableBody");
    const emptyHint = document.getElementById("emptyHint");
    const refreshButton = document.getElementById("refreshButton");
    const runningLabel = {{printf "%q" .RunningLabel}};
    const errorPrefix = {{printf "%q" .ErrorPrefix}};
    const okLabel = {{printf "%q" .OKLabel}};
    const installLabel = {{printf "%q" .InstallButton}};
    const uninstallLabel = {{printf "%q" .UninstallButton}};
    const checkLabel = {{printf "%q" .CheckButton}};
    const statusInstalled = {{printf "%q" .StatusInstalled}};
    const statusNotInstalled = {{printf "%q" .StatusNotInstalled}};
    const statusUnknown = {{printf "%q" .StatusUnknown}};
    const statusLoading = {{printf "%q" .StatusLoading}};
    const statusError = {{printf "%q" .StatusError}};
    const statusReady = {{printf "%q" .StatusReady}};

    const endpoints = {
      list: "/api/v1/modules",
      install: "/api/v1/module/install",
      uninstall: "/api/v1/module/uninstall",
      check: "/api/v1/module/check",
    };

    function clearTable() {
      while (tableBody.firstChild) {
        tableBody.removeChild(tableBody.firstChild);
      }
    }

    function cleanText(value) {
      if (typeof value !== "string") {
        return value || "";
      }
      const trimmed = value.trim();
      if (trimmed.length >= 2) {
        const first = trimmed[0];
        const last = trimmed[trimmed.length - 1];
        if ((first === "\"" && last === "\"") || (first === "'" && last === "'")) {
          return trimmed.slice(1, -1).trim();
        }
      }
      return trimmed;
    }

    function statusLabel(status) {
      if (status === "installed") {
        return cleanText(statusInstalled);
      }
      if (status === "not_installed") {
        return cleanText(statusNotInstalled);
      }
      return cleanText(statusUnknown);
    }

    function renderModules(modules) {
      clearTable();
      if (!modules || modules.length === 0) {
        emptyHint.style.display = "block";
        return;
      }
      emptyHint.style.display = "none";
      modules.forEach((module) => {
        const row = document.createElement("tr");

        const nameCell = document.createElement("td");
        nameCell.textContent = cleanText(module.name);
        row.appendChild(nameCell);

        const categoryCell = document.createElement("td");
        categoryCell.textContent = cleanText(module.category);
        row.appendChild(categoryCell);

        const versionCell = document.createElement("td");
        versionCell.textContent = cleanText(module.version);
        row.appendChild(versionCell);

        const statusCell = document.createElement("td");
        statusCell.textContent = statusLabel(module.status);
        row.appendChild(statusCell);

        const actionsCell = document.createElement("td");
        actionsCell.className = "actions";

        actionsCell.appendChild(actionButton("install", module.name, installLabel));
        actionsCell.appendChild(actionButton("uninstall", module.name, uninstallLabel));
        actionsCell.appendChild(actionButton("check", module.name, checkLabel));

        row.appendChild(actionsCell);
        tableBody.appendChild(row);
      });
    }

    function actionButton(action, name, label) {
      const button = document.createElement("button");
      button.textContent = cleanText(label);
      button.className = action === "install" ? "primary" : (action === "uninstall" ? "secondary" : "ghost");
      button.addEventListener("click", () => runAction(action, name));
      return button;
    }

    async function fetchModules() {
        statusText.textContent = cleanText(statusLoading);
      try {
        const resp = await fetch(endpoints.list, { method: "GET" });
        const data = await resp.json();
        if (!resp.ok || !data.ok) {
          const message = data && data.error ? data.error : resp.statusText;
          statusText.textContent = cleanText(statusError) + ": " + message;
          renderModules([]);
          return;
        }
        renderModules(data.modules || []);
        statusText.textContent = cleanText(statusReady);
      } catch (err) {
        statusText.textContent = cleanText(statusError) + ": " + err.message;
      }
    }

    async function runAction(action, name) {
      statusText.textContent = cleanText(runningLabel);
      const payload = { name: name };

      try {
        const resp = await fetch(endpoints[action], {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });

        const data = await resp.json();
        if (!resp.ok || !data.ok) {
          const message = data && data.error ? data.error : resp.statusText;
          statusText.textContent = cleanText(errorPrefix) + ": " + message;
          return;
        }
        statusText.textContent = cleanText(okLabel);
        fetchModules();
      } catch (err) {
        statusText.textContent = cleanText(errorPrefix) + ": " + err.message;
      }
    }

    refreshButton.addEventListener("click", () => fetchModules());
    fetchModules();
  </script>
</body>
</html>`
