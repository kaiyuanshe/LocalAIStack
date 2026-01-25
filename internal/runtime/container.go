package runtime

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

func (m *Manager) startContainer(ctx context.Context, spec ModuleSpec, proc *process) error {
	containerBin, err := m.resolveContainerRuntime(spec.ContainerRuntime)
	if err != nil {
		return err
	}
	name := spec.ContainerName
	if name == "" {
		name = fmt.Sprintf("%s-%d", spec.Name, time.Now().Unix())
	}
	if spec.Image == "" {
		return i18n.Errorf("container image is required")
	}

	args := []string{"run", "-d", "--rm", "--name", name}
	for key, value := range spec.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}
	if spec.WorkDir != "" {
		args = append(args, "-w", spec.WorkDir)
	}
	args = append(args, spec.Image)
	if len(spec.Command) > 0 {
		args = append(args, spec.Command...)
	}
	if len(spec.Args) > 0 {
		args = append(args, spec.Args...)
	}

	cmd := exec.CommandContext(ctx, containerBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return i18n.Errorf("start container: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	containerID := strings.TrimSpace(string(output))
	if containerID == "" {
		return i18n.Errorf("container runtime did not return container id")
	}

	proc.status.State = StateRunning
	proc.status.ContainerID = containerID
	proc.containerID = containerID
	proc.containerBin = containerBin

	m.startContainerLogs(proc)
	go m.watchContainer(proc)
	return nil
}

func (m *Manager) startContainerLogs(proc *process) {
	ctx, cancel := context.WithCancel(context.Background())
	proc.cancelLogs = cancel
	logCmd := exec.CommandContext(ctx, proc.containerBin, "logs", "-f", proc.containerID)
	logCmd.Stdout = proc.logFile
	logCmd.Stderr = proc.logFile
	if err := logCmd.Start(); err != nil {
		proc.status.LastError = i18n.T("start log stream: %s", err)
		return
	}
	go func() {
		_ = logCmd.Wait()
		m.closeLogFile(proc)
	}()
}

func (m *Manager) watchContainer(proc *process) {
	cmd := exec.Command(proc.containerBin, "wait", proc.containerID)
	if err := cmd.Run(); err != nil {
		m.markStopped(proc, err)
		m.stopLogStream(proc)
		return
	}
	m.markStopped(proc, nil)
	m.stopLogStream(proc)
}

func (m *Manager) stopContainer(ctx context.Context, proc *process) error {
	if proc.containerBin == "" || proc.containerID == "" {
		return nil
	}
	cmd := exec.CommandContext(ctx, proc.containerBin, "stop", "-t", "10", proc.containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return i18n.Errorf("stop container: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	m.markStopped(proc, nil)
	m.stopLogStream(proc)
	return nil
}

func (m *Manager) resolveContainerRuntime(preferred string) (string, error) {
	if !m.dockerEnabled {
		return "", i18n.Errorf("container runtime disabled")
	}
	if preferred != "" {
		if _, err := exec.LookPath(preferred); err == nil {
			return preferred, nil
		}
		return "", i18n.Errorf("container runtime %q not found", preferred)
	}
	for _, candidate := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", i18n.Errorf("no container runtime found")
}
