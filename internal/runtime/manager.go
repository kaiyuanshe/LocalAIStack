package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zhuangbiaowei/LocalAIStack/internal/config"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

type Manager struct {
	baseDir       string
	defaultMode   ExecutionMode
	dockerEnabled bool
	nativeEnabled bool
	mu            sync.Mutex
	processes     map[string]*process
}

type process struct {
	status       Status
	cmd          *exec.Cmd
	containerID  string
	containerBin string
	cancelRun    context.CancelFunc
	cancelLogs   context.CancelFunc
	cancelHealth context.CancelFunc
	logFile      *os.File
	healthCheck  HealthCheck
}

func NewManager(cfg config.RuntimeConfig) *Manager {
	baseDir := cfg.LogDir
	if baseDir == "" {
		baseDir = "/var/lib/localaistack/runtime"
	}
	mode := ExecutionMode(cfg.DefaultMode)
	if mode == "" {
		mode = ModeContainer
	}
	return &Manager{
		baseDir:       baseDir,
		defaultMode:   mode,
		dockerEnabled: cfg.DockerEnabled,
		nativeEnabled: cfg.NativeEnabled,
		processes:     make(map[string]*process),
	}
}

func (m *Manager) Start(ctx context.Context, spec ModuleSpec) (*Status, error) {
	if strings.TrimSpace(spec.Name) == "" {
		return nil, i18n.Errorf("module name is required")
	}
	mode := spec.Mode
	if mode == "" {
		mode = m.defaultMode
	}
	if err := m.validateMode(mode); err != nil {
		return nil, err
	}

	m.mu.Lock()
	if existing, ok := m.processes[spec.Name]; ok && existing.status.State == StateRunning {
		m.mu.Unlock()
		return nil, i18n.Errorf("module %q already running", spec.Name)
	}
	m.mu.Unlock()

	logFile, logPath, err := m.createLogFile(spec.Name)
	if err != nil {
		return nil, err
	}

	status := Status{
		Name:      spec.Name,
		Mode:      mode,
		State:     StateStarting,
		Health:    HealthUnknown,
		StartedAt: time.Now(),
		LogPath:   logPath,
	}

	proc := &process{
		status:      status,
		logFile:     logFile,
		healthCheck: spec.HealthCheck,
	}

	switch mode {
	case ModeNative:
		if err := m.startNative(ctx, spec, proc); err != nil {
			m.closeLogFile(proc)
			return nil, err
		}
	case ModeContainer:
		if err := m.startContainer(ctx, spec, proc); err != nil {
			m.closeLogFile(proc)
			return nil, err
		}
	default:
		m.closeLogFile(proc)
		return nil, i18n.Errorf("unsupported execution mode: %s", mode)
	}

	m.mu.Lock()
	m.processes[spec.Name] = proc
	m.mu.Unlock()

	m.startHealthMonitor(proc)

	return &proc.status, nil
}

func (m *Manager) Stop(ctx context.Context, name string) error {
	proc, ok := m.getProcess(name)
	if !ok {
		return i18n.Errorf("module %q not found", name)
	}

	var err error
	switch proc.status.Mode {
	case ModeNative:
		err = m.stopNative(ctx, proc)
	case ModeContainer:
		err = m.stopContainer(ctx, proc)
	default:
		err = i18n.Errorf("unsupported execution mode: %s", proc.status.Mode)
	}

	m.stopHealthMonitor(proc)
	return err
}

func (m *Manager) Status(name string) (Status, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	proc, ok := m.processes[name]
	if !ok {
		return Status{}, false
	}
	return proc.status, true
}

func (m *Manager) List() []Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	statuses := make([]Status, 0, len(m.processes))
	for _, proc := range m.processes {
		statuses = append(statuses, proc.status)
	}
	return statuses
}

func (m *Manager) validateMode(mode ExecutionMode) error {
	switch mode {
	case ModeContainer:
		if !m.dockerEnabled {
			return i18n.Errorf("container runtime disabled")
		}
	case ModeNative:
		if !m.nativeEnabled {
			return i18n.Errorf("native runtime disabled")
		}
	default:
		return i18n.Errorf("invalid runtime mode %q", mode)
	}
	return nil
}

func (m *Manager) createLogFile(name string) (*os.File, string, error) {
	timestamp := time.Now().UTC().Format("20060102-150405")
	logDir := filepath.Join(m.baseDir, "logs", name)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, "", i18n.Errorf("create log dir: %w", err)
	}
	logPath := filepath.Join(logDir, fmt.Sprintf("%s.log", timestamp))
	file, err := os.Create(logPath)
	if err != nil {
		return nil, "", i18n.Errorf("create log file: %w", err)
	}
	return file, logPath, nil
}

func (m *Manager) closeLogFile(proc *process) {
	if proc.logFile != nil {
		if err := proc.logFile.Close(); err != nil {
			log.Warn().Err(err).Msg(i18n.T("failed to close log file"))
		}
	}
}

func (m *Manager) getProcess(name string) (*process, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	proc, ok := m.processes[name]
	return proc, ok
}

func (m *Manager) updateProcess(name string, update func(*process)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	proc, ok := m.processes[name]
	if !ok {
		return
	}
	update(proc)
}

func (m *Manager) markStopped(proc *process, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if proc.status.State == StateStopped || proc.status.State == StateFailed {
		return
	}
	finished := time.Now()
	proc.status.FinishedAt = &finished
	if err != nil {
		proc.status.State = StateFailed
		proc.status.LastError = err.Error()
	} else {
		proc.status.State = StateStopped
	}
	proc.status.Health = HealthUnhealthy
}

func (m *Manager) startHealthMonitor(proc *process) {
	if proc.healthCheck.Interval == 0 {
		proc.healthCheck.Interval = 30 * time.Second
	}
	if proc.healthCheck.Timeout == 0 {
		proc.healthCheck.Timeout = 5 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	proc.cancelHealth = cancel
	go func() {
		ticker := time.NewTicker(proc.healthCheck.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				health := m.checkHealth(proc)
				m.updateProcess(proc.status.Name, func(p *process) {
					p.status.Health = health
				})
			}
		}
	}()
}

func (m *Manager) stopHealthMonitor(proc *process) {
	if proc.cancelHealth != nil {
		proc.cancelHealth()
	}
}

func (m *Manager) checkHealth(proc *process) HealthState {
	switch proc.status.Mode {
	case ModeNative:
		return m.checkNativeHealth(proc)
	case ModeContainer:
		return m.checkContainerHealth(proc)
	default:
		return HealthUnknown
	}
}

func (m *Manager) checkNativeHealth(proc *process) HealthState {
	if len(proc.healthCheck.Command) == 0 {
		if proc.status.State == StateRunning {
			return HealthHealthy
		}
		if proc.status.State == StateFailed || proc.status.State == StateStopped {
			return HealthUnhealthy
		}
		return HealthUnknown
	}

	ctx, cancel := context.WithTimeout(context.Background(), proc.healthCheck.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, proc.healthCheck.Command[0], proc.healthCheck.Command[1:]...)
	if err := cmd.Run(); err != nil {
		return HealthUnhealthy
	}
	return HealthHealthy
}

func (m *Manager) checkContainerHealth(proc *process) HealthState {
	if proc.containerBin == "" || proc.containerID == "" {
		return HealthUnknown
	}
	if len(proc.healthCheck.Command) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), proc.healthCheck.Timeout)
		defer cancel()
		args := append([]string{"exec", proc.containerID}, proc.healthCheck.Command...)
		cmd := exec.CommandContext(ctx, proc.containerBin, args...)
		if err := cmd.Run(); err != nil {
			return HealthUnhealthy
		}
		return HealthHealthy
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, proc.containerBin, "inspect", "--format", "{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}", proc.containerID)
	output, err := cmd.Output()
	if err != nil {
		return HealthUnknown
	}
	status := strings.TrimSpace(strings.ToLower(string(output)))
	switch status {
	case "healthy", "running":
		return HealthHealthy
	case "unhealthy", "exited", "dead":
		return HealthUnhealthy
	default:
		return HealthUnknown
	}
}

func (m *Manager) stopLogStream(proc *process) {
	if proc.cancelLogs != nil {
		proc.cancelLogs()
	}
}

func (m *Manager) waitForExit(proc *process, waitFunc func() error) {
	if err := waitFunc(); err != nil {
		m.markStopped(proc, err)
	} else {
		m.markStopped(proc, nil)
	}
	m.stopLogStream(proc)
	m.stopHealthMonitor(proc)
	m.closeLogFile(proc)
}

func buildEnv(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	merged := os.Environ()
	for key, value := range env {
		merged = append(merged, fmt.Sprintf("%s=%s", key, value))
	}
	return merged
}

func joinCommand(command []string, args []string) ([]string, error) {
	if len(command) == 0 {
		return nil, errors.New("command is required")
	}
	full := append([]string{}, command...)
	full = append(full, args...)
	return full, nil
}
