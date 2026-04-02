package module

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

func ensurePlatformSupported(platforms []string) error {
	if len(platforms) == 0 {
		return nil
	}
	current := runtime.GOOS + "/" + runtime.GOARCH
	for _, platform := range platforms {
		if strings.EqualFold(strings.TrimSpace(platform), current) {
			return nil
		}
	}
	return i18n.Errorf("current platform %s is not supported by this module", current)
}

func resolveModuleScriptPath(scriptPath string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		if strings.EqualFold(filepath.Ext(scriptPath), ".sh") {
			ps1Path := strings.TrimSuffix(scriptPath, filepath.Ext(scriptPath)) + ".ps1"
			if _, err := os.Stat(ps1Path); err == nil {
				return ps1Path, nil
			}
		}
	default:
		if strings.EqualFold(filepath.Ext(scriptPath), ".ps1") {
			shPath := strings.TrimSuffix(scriptPath, filepath.Ext(scriptPath)) + ".sh"
			if _, err := os.Stat(shPath); err == nil {
				return shPath, nil
			}
		}
	}

	if _, err := os.Stat(scriptPath); err != nil {
		return "", err
	}
	return scriptPath, nil
}

func newModuleScriptCommand(scriptPath, moduleDir string, scriptArgs []string, env map[string]string) (*exec.Cmd, error) {
	resolvedPath, err := resolveModuleScriptPath(scriptPath)
	if err != nil {
		return nil, err
	}

	var cmd *exec.Cmd
	switch strings.ToLower(filepath.Ext(resolvedPath)) {
	case ".ps1":
		args := append([]string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", resolvedPath}, scriptArgs...)
		cmd = exec.Command("powershell.exe", args...)
	case ".sh":
		args := append([]string{resolvedPath}, scriptArgs...)
		cmd = exec.Command("bash", args...)
	default:
		args := append([]string{resolvedPath}, scriptArgs...)
		cmd = exec.Command(resolvedPath, args...)
	}

	cmd.Dir = moduleDir
	cmd.Env = commandEnv(env)
	return cmd, nil
}

func runModuleScript(scriptPath, moduleDir string, scriptArgs []string, env map[string]string, stream bool) (string, error) {
	cmd, err := newModuleScriptCommand(scriptPath, moduleDir, scriptArgs, env)
	if err != nil {
		return "", err
	}

	if stream {
		var buffer bytes.Buffer
		writer := io.MultiWriter(&buffer, os.Stdout)
		cmd.Stdout = writer
		cmd.Stderr = writer
		if err := cmd.Run(); err != nil {
			message := normalizedOutput(buffer.String())
			if message == "" {
				return "", err
			}
			return message, i18n.Errorf("%s", message)
		}
		return buffer.String(), nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		message := normalizedOutput(string(output))
		if message == "" {
			return "", err
		}
		return message, i18n.Errorf("%s", message)
	}
	return string(output), nil
}

func runShellCommand(command, moduleDir string, stream bool) (string, int, error) {
	return runShellCommandWithEnv(command, moduleDir, stream, nil)
}

func runShellCommandWithEnv(command, moduleDir string, stream bool, env map[string]string) (string, int, error) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "", 0, i18n.Errorf("empty shell command")
	}

	if scriptPath, scriptArgs, ok := parseScriptShellCommand(trimmed); ok {
		output, err := runModuleScript(absoluteModuleScriptPath(moduleDir, scriptPath), moduleDir, scriptArgs, env, stream)
		exitCode := 0
		if err != nil {
			return output, 1, err
		}
		return output, exitCode, nil
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", trimmed)
	default:
		cmd = exec.Command("bash", "-c", trimmed)
	}

	cmd.Dir = moduleDir
	cmd.Env = commandEnv(env)
	if stream {
		var buffer bytes.Buffer
		writer := io.MultiWriter(&buffer, os.Stdout)
		cmd.Stdout = writer
		cmd.Stderr = writer
		err := cmd.Run()
		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		output := buffer.String()
		if err != nil {
			message := normalizedOutput(output)
			if message == "" {
				return "", exitCode, err
			}
			return message, exitCode, i18n.Errorf("%s", message)
		}
		return output, exitCode, nil
	}

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		message := normalizedOutput(string(output))
		if message == "" {
			return "", exitCode, err
		}
		return message, exitCode, i18n.Errorf("%s", message)
	}
	return string(output), exitCode, nil
}

var bashScriptPattern = regexp.MustCompile(`^(?:bash|sh)\s+("?[^"\s]+\.sh"?)(?:\s+(.*))?$`)

func parseScriptShellCommand(command string) (string, []string, bool) {
	matches := bashScriptPattern.FindStringSubmatch(strings.TrimSpace(command))
	if len(matches) == 0 {
		return "", nil, false
	}

	scriptPath := strings.Trim(matches[1], `"`)
	var args []string
	if len(matches) >= 3 && strings.TrimSpace(matches[2]) != "" {
		args = splitShellLikeArgs(matches[2])
	}
	return scriptPath, args, true
}

func splitShellLikeArgs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var args []string
	var current strings.Builder
	var quote rune
	escaped := false

	flush := func() {
		if current.Len() == 0 {
			return
		}
		args = append(args, current.String())
		current.Reset()
	}

	for _, ch := range raw {
		switch {
		case escaped:
			current.WriteRune(ch)
			escaped = false
		case ch == '\\' && quote != '\'':
			escaped = true
		case quote != 0:
			if ch == quote {
				quote = 0
			} else {
				current.WriteRune(ch)
			}
		case ch == '\'' || ch == '"':
			quote = ch
		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			flush()
		default:
			current.WriteRune(ch)
		}
	}
	flush()
	return args
}

func platformCheckScriptPath(moduleDir string) string {
	return filepath.Join(moduleDir, "scripts", "verify.sh")
}

func platformSettingScriptPath(moduleDir string) string {
	return filepath.Join(moduleDir, "scripts", "setting.sh")
}

func absoluteModuleScriptPath(moduleDir, script string) string {
	if filepath.IsAbs(script) {
		return script
	}
	return filepath.Join(moduleDir, script)
}

func formatResolvedScriptPath(moduleDir, script string) string {
	path, err := resolveModuleScriptPath(absoluteModuleScriptPath(moduleDir, script))
	if err != nil {
		return fmt.Sprintf("%s (%v)", script, err)
	}
	return path
}
