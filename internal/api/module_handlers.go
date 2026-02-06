package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/module"
)

type moduleRequest struct {
	Name string `json:"name"`
}

type cliResult struct {
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
}

type cliResponse struct {
	OK     bool       `json:"ok"`
	Error  string     `json:"error,omitempty"`
	Output string     `json:"output,omitempty"`
	Result *cliResult `json:"result,omitempty"`
}

type moduleInfo struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type modulesResponse struct {
	OK      bool         `json:"ok"`
	Error   string       `json:"error,omitempty"`
	Modules []moduleInfo `json:"modules,omitempty"`
}

func (s *Server) modulesListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, i18n.T("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	if _, err := s.runCLI(r.Context(), []string{"module", "list"}); err != nil {
		writeModulesResponse(w, nil, err)
		return
	}

	modules, err := listModules()
	writeModulesResponse(w, modules, err)
}

func (s *Server) moduleListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, i18n.T("method not allowed"), http.StatusMethodNotAllowed)
		return
	}
	result, err := s.runCLI(r.Context(), []string{"module", "list"})
	writeCLIResponse(w, result, err)
}

func (s *Server) moduleInstallHandler(w http.ResponseWriter, r *http.Request) {
	s.moduleActionHandler(w, r, "install")
}

func (s *Server) moduleUninstallHandler(w http.ResponseWriter, r *http.Request) {
	s.moduleActionHandler(w, r, "uninstall")
}

func (s *Server) moduleCheckHandler(w http.ResponseWriter, r *http.Request) {
	s.moduleActionHandler(w, r, "check")
}

func (s *Server) moduleActionHandler(w http.ResponseWriter, r *http.Request, action string) {
	if r.Method != http.MethodPost {
		http.Error(w, i18n.T("method not allowed"), http.StatusMethodNotAllowed)
		return
	}

	var req moduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeCLIResponse(w, nil, fmt.Errorf("invalid request body: %w", err))
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeCLIResponse(w, nil, errors.New("module name cannot be empty"))
		return
	}

	args := []string{"module", action, name}
	result, err := s.runCLI(r.Context(), args)
	writeCLIResponse(w, result, err)
}

func (s *Server) runCLI(ctx context.Context, args []string) (*cliResult, error) {
	cliPath, err := findCLIPath()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, cliPath, args...)
	cmd.Env = os.Environ()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	cmd.Stdin = os.Stdin

	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr := new(exec.ExitError); errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, err
		}
	}

	return &cliResult{
		ExitCode:   exitCode,
		Stdout:     strings.TrimSpace(stdoutBuf.String()),
		Stderr:     strings.TrimSpace(stderrBuf.String()),
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func findCLIPath() (string, error) {
	if env := strings.TrimSpace(os.Getenv("LAS_CLI_PATH")); env != "" {
		if fileExists(env) {
			return env, nil
		}
		return "", fmt.Errorf("LAS_CLI_PATH not found: %s", env)
	}

	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		path := filepath.Join(dir, "las")
		if fileExists(path) {
			return path, nil
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		candidates := []string{
			filepath.Join(cwd, "build", "las"),
			filepath.Join(cwd, "las"),
		}
		for _, candidate := range candidates {
			if fileExists(candidate) {
				return candidate, nil
			}
		}
	}

	if path, err := exec.LookPath("las"); err == nil {
		return path, nil
	}
	if path, err := exec.LookPath("localaistack"); err == nil {
		return path, nil
	}

	return "", errors.New("las CLI not found; build it first with `make build` or set LAS_CLI_PATH")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func writeCLIResponse(w http.ResponseWriter, result *cliResult, err error) {
	w.Header().Set("Content-Type", "application/json")
	response := cliResponse{}
	if err != nil {
		response.OK = false
		response.Error = err.Error()
		if result != nil {
			response.Output = combineOutput(result)
			response.Result = result
		}
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(response)
		return
	}

	response.OK = true
	response.Result = result
	if result != nil {
		response.Output = combineOutput(result)
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func combineOutput(result *cliResult) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(result.Stdout) != "" {
		parts = append(parts, result.Stdout)
	}
	if strings.TrimSpace(result.Stderr) != "" {
		parts = append(parts, result.Stderr)
	}
	return strings.Join(parts, "\n")
}

func listModules() ([]moduleInfo, error) {
	modulesRoot, err := module.FindModulesRoot()
	if err != nil {
		return nil, err
	}
	registry, err := module.LoadRegistryFromDir(modulesRoot)
	if err != nil {
		return nil, err
	}

	all := registry.All()
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]moduleInfo, 0, len(names))
	for _, name := range names {
		records := all[name]
		if len(records) == 0 {
			continue
		}
		record := records[0]
		status := "not_installed"
		if err := module.Check(name); err == nil {
			status = "installed"
		}
		result = append(result, moduleInfo{
			Name:        name,
			Category:    string(record.Manifest.Category),
			Version:     record.Manifest.Version,
			Description: record.Manifest.Description,
			Status:      status,
		})
	}
	return result, nil
}

func writeModulesResponse(w http.ResponseWriter, modules []moduleInfo, err error) {
	w.Header().Set("Content-Type", "application/json")
	response := modulesResponse{}
	if err != nil {
		response.OK = false
		response.Error = err.Error()
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(response)
		return
	}
	response.OK = true
	response.Modules = modules
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
