package module

import (
	"context"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

const ollamaTagsURL = "http://127.0.0.1:11434/api/tags"

func Check(name string) error {
	switch strings.ToLower(name) {
	case "ollama":
		return checkOllama()
	default:
		return i18n.Errorf("unsupported module %q", name)
	}
}

func checkOllama() error {
	if _, err := exec.LookPath("ollama"); err != nil {
		return i18n.Errorf("ollama CLI not found in PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	versionCmd := exec.CommandContext(ctx, "ollama", "--version")
	if err := versionCmd.Run(); err != nil {
		return i18n.Errorf("ollama CLI not responding: %w", err)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(ollamaTagsURL)
	if err != nil {
		return i18n.Errorf("ollama service not responding on %s: %w", ollamaTagsURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return i18n.Errorf("ollama service returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return i18n.Errorf("read ollama response: %w", err)
	}
	if !strings.Contains(string(body), "\"models\"") {
		return i18n.Errorf("ollama response missing models data")
	}
	return nil
}
