package modelmanager

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ollamaAPIURL     = "https://ollama.com/api"
	ollamaAPITimeout = 30 * time.Second
)

type OllamaProvider struct {
	client *http.Client
}

func NewOllamaProvider() *OllamaProvider {
	return &OllamaProvider{
		client: &http.Client{Timeout: ollamaAPITimeout},
	}
}

func (p *OllamaProvider) Name() ModelSource {
	return SourceOllama
}

type OllamaAPIModel struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
	Details    struct {
		ParentModel       string   `json:"parent_model"`
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

type OllamaTagsResponse struct {
	Models []OllamaAPIModel `json:"models"`
}

func (p *OllamaProvider) Search(ctx context.Context, query string, limit int) ([]ModelInfo, error) {
	url := fmt.Sprintf("%s/tags", ollamaAPIURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search Ollama models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tagsResp OllamaTagsResponse
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	var models []ModelInfo
	queryLower := strings.ToLower(query)
	count := 0

	for _, om := range tagsResp.Models {
		if limit > 0 && count >= limit {
			break
		}

		if query != "" && !strings.Contains(strings.ToLower(om.Name), queryLower) {
			continue
		}

		models = append(models, ModelInfo{
			ID:          om.Name,
			Name:        om.Name,
			Description: fmt.Sprintf("Size: %s, Family: %s", FormatBytes(om.Size), om.Details.Family),
			Source:      SourceOllama,
			Format:      FormatOllama,
			Size:        om.Size,
			Tags:        om.Details.Families,
			Metadata: map[string]string{
				"digest":             om.Digest,
				"modified_at":        om.ModifiedAt,
				"parameter_size":     om.Details.ParameterSize,
				"quantization_level": om.Details.QuantizationLevel,
			},
		})
		count++
	}

	return models, nil
}

func (p *OllamaProvider) Download(ctx context.Context, modelID string, destPath string, progress func(downloaded, total int64)) error {
	modelDir := filepath.Join(destPath, modelID)
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	if err := p.pullModel(ctx, modelID, progress); err != nil {
		return fmt.Errorf("failed to pull Ollama model: %w", err)
	}

	metadata := map[string]interface{}{
		"id":        modelID,
		"source":    "ollama",
		"format":    "ollama",
		"pulled_at": time.Now().Unix(),
	}

	metadataPath := filepath.Join(modelDir, "metadata.json")
	metadataFile, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to create metadata file: %w", err)
	}
	defer metadataFile.Close()

	encoder := json.NewEncoder(metadataFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

func (p *OllamaProvider) pullModel(ctx context.Context, modelID string, progress func(downloaded, total int64)) error {
	cmd := exec.CommandContext(ctx, "ollama", "pull", modelID)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ollama pull: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if progress != nil {
				progress(0, 0)
			}
			_ = line
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			_ = scanner.Text()
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ollama pull failed: %w", err)
	}

	return nil
}

func (p *OllamaProvider) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
	url := fmt.Sprintf("%s/tags", ollamaAPIURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get model info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tagsResp OllamaTagsResponse
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return nil, fmt.Errorf("failed to parse model info: %w", err)
	}

	for _, om := range tagsResp.Models {
		if om.Name == modelID {
			return &ModelInfo{
				ID:          om.Name,
				Name:        om.Name,
				Description: fmt.Sprintf("Size: %s, Family: %s", FormatBytes(om.Size), om.Details.Family),
				Source:      SourceOllama,
				Format:      FormatOllama,
				Size:        om.Size,
				Tags:        om.Details.Families,
				Metadata: map[string]string{
					"digest":             om.Digest,
					"modified_at":        om.ModifiedAt,
					"parameter_size":     om.Details.ParameterSize,
					"quantization_level": om.Details.QuantizationLevel,
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("model %s not found", modelID)
}

func (p *OllamaProvider) Delete(ctx context.Context, modelID string) error {
	cmd := exec.CommandContext(ctx, "ollama", "rm", modelID)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete Ollama model %s: %w (output: %s)", modelID, err, string(output))
	}

	return nil
}
