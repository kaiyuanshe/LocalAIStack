package system

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type BaseInfo struct {
	CollectedAt string `json:"collected_at"`
	Hostname    string `json:"hostname"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	CPUs        int    `json:"cpus"`
	GoVersion   string `json:"go_version"`
	User        string `json:"user"`
	HomeDir     string `json:"home_dir"`
}

func WriteBaseInfo(outputPath, format string, force, appendMode bool) error {
	resolvedPath, err := expandHome(outputPath)
	if err != nil {
		return err
	}
	if err := ensureWritable(resolvedPath, force, appendMode); err != nil {
		return err
	}
	info, err := collectBaseInfo()
	if err != nil {
		return err
	}

	content, err := formatBaseInfo(info, format)
	if err != nil {
		return err
	}

	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(resolvedPath, flags, 0o644)
	if err != nil {
		return fmt.Errorf("open output file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	return nil
}

func collectBaseInfo() (BaseInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return BaseInfo{}, fmt.Errorf("get hostname: %w", err)
	}

	currentUser, err := user.Current()
	if err != nil {
		return BaseInfo{}, fmt.Errorf("get current user: %w", err)
	}

	return BaseInfo{
		CollectedAt: time.Now().Format(time.RFC3339),
		Hostname:    hostname,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		CPUs:        runtime.NumCPU(),
		GoVersion:   runtime.Version(),
		User:        currentUser.Username,
		HomeDir:     currentUser.HomeDir,
	}, nil
}

func formatBaseInfo(info BaseInfo, format string) (string, error) {
	switch strings.ToLower(format) {
	case "md", "markdown":
		return fmt.Sprintf(`# LocalAIStack Base Info

- Collected At: %s
- Hostname: %s
- OS: %s
- Arch: %s
- CPUs: %d
- Go Version: %s
- User: %s
- Home Directory: %s
`, info.CollectedAt, info.Hostname, info.OS, info.Arch, info.CPUs, info.GoVersion, info.User, info.HomeDir), nil
	case "json":
		payload, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return "", fmt.Errorf("marshal json: %w", err)
		}
		return string(payload) + "\n", nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func expandHome(path string) (string, error) {
	if path == "" {
		return "", errors.New("output path is required")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func ensureWritable(path string, force, appendMode bool) error {
	if path == "" {
		return errors.New("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if appendMode {
		return nil
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("output file exists: %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat output file: %w", err)
		}
	}
	return nil
}
