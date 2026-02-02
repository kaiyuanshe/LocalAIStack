package modelmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	modelscopeAPIURL   = "https://www.modelscope.cn/api/v1"
	modelscopeModelURL = "https://www.modelscope.cn/models"
	modelscopeTimeout  = 60 * time.Second
)

type ModelScopeProvider struct {
	client *http.Client
	token  string
}

func NewModelScopeProvider(token string) *ModelScopeProvider {
	return &ModelScopeProvider{
		client: &http.Client{Timeout: modelscopeTimeout},
		token:  token,
	}
}

func (p *ModelScopeProvider) Name() ModelSource {
	return SourceModelScope
}

type ModelScopeModel struct {
	ModelID     string   `json:"ModelId"`
	Name        string   `json:"Name"`
	Description string   `json:"Description"`
	Tags        []string `json:"Tags"`
	Downloads   int      `json:"Downloads"`
	Likes       int      `json:"Likes"`
	Visibility  string   `json:"Visibility"`
}

type ModelScopeFile struct {
	Path string `json:"Path"`
	Size int64  `json:"Size"`
	Type string `json:"Type"`
}

type ModelScopeSearchResponse struct {
	Data struct {
		Models []ModelScopeModel `json:"Models"`
	} `json:"Data"`
}

type ModelScopeFilesResponse struct {
	Data struct {
		Files []ModelScopeFile `json:"Files"`
	} `json:"Data"`
}

func (p *ModelScopeProvider) Search(ctx context.Context, query string, limit int) ([]ModelInfo, error) {
	if limit <= 0 {
		limit = 20
	}

	url := fmt.Sprintf("%s/models?search=%s&page_size=%d", modelscopeAPIURL, query, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "LocalAIStack/1.0")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search ModelScope models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("ModelScope API access forbidden - the API may require authentication or have CORS restrictions")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ModelScope API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp ModelScopeSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse ModelScope response: %w", err)
	}

	var models []ModelInfo
	for _, mm := range searchResp.Data.Models {
		format := p.detectFormatFromTags(mm.Tags)

		models = append(models, ModelInfo{
			ID:          mm.ModelID,
			Name:        mm.Name,
			Description: mm.Description,
			Source:      SourceModelScope,
			Format:      format,
			Tags:        mm.Tags,
			Metadata: map[string]string{
				"downloads":  fmt.Sprintf("%d", mm.Downloads),
				"likes":      fmt.Sprintf("%d", mm.Likes),
				"visibility": mm.Visibility,
			},
		})
	}

	return models, nil
}

func (p *ModelScopeProvider) detectFormatFromTags(tags []string) ModelFormat {
	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		if strings.Contains(tagLower, "gguf") {
			return FormatGGUF
		}
		if strings.Contains(tagLower, "safetensors") {
			return FormatSafetensors
		}
	}
	return FormatUnknown
}

func (p *ModelScopeProvider) Download(ctx context.Context, modelID string, destPath string, progress func(downloaded, total int64), opts DownloadOptions) error {
	files, err := p.listModelFiles(ctx, modelID)
	if err != nil {
		return fmt.Errorf("failed to list model files: %w", err)
	}

	modelDir := filepath.Join(destPath, strings.ReplaceAll(modelID, "/", "_"))
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	candidates, err := filterModelScopeFiles(files, opts.FileHint)
	if err != nil {
		return err
	}

	for _, file := range candidates {
		fileURL := fmt.Sprintf("%s/models/%s/repo?file_path=%s", modelscopeAPIURL, modelID, file.Path)
		destFile := filepath.Join(modelDir, filepath.Base(file.Path))

		if err := p.downloadFile(ctx, fileURL, destFile, file.Size, progress); err != nil {
			return fmt.Errorf("failed to download file %s: %w", file.Path, err)
		}
	}

	metadata := map[string]interface{}{
		"id":            modelID,
		"source":        "modelscope",
		"downloaded_at": time.Now().Unix(),
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

func filterModelScopeFiles(files []ModelScopeFile, hint string) ([]ModelScopeFile, error) {
	allowed := make([]ModelScopeFile, 0, len(files))
	for _, file := range files {
		if file.Type != "file" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Path))
		if ext != ".gguf" && ext != ".safetensors" && ext != ".bin" {
			continue
		}
		allowed = append(allowed, file)
	}

	if hint == "" {
		return allowed, nil
	}

	normalized := strings.ToLower(strings.TrimSpace(hint))
	if normalized == "" {
		return allowed, nil
	}

	exact := make([]ModelScopeFile, 0)
	contains := make([]ModelScopeFile, 0)
	for _, file := range allowed {
		base := strings.ToLower(filepath.Base(file.Path))
		if base == normalized {
			exact = append(exact, file)
			continue
		}
		if strings.Contains(base, normalized) || strings.Contains(strings.ToLower(file.Path), normalized) {
			contains = append(contains, file)
		}
	}

	if len(exact) == 1 {
		return exact, nil
	}
	if len(exact) > 1 {
		return nil, fmt.Errorf("multiple files match %q; please specify a more specific filename", hint)
	}
	if len(contains) == 1 {
		return contains, nil
	}
	if len(contains) > 1 {
		names := make([]string, 0, len(contains))
		for _, file := range contains {
			names = append(names, filepath.Base(file.Path))
		}
		sort.Strings(names)
		return nil, fmt.Errorf("multiple files match %q: %s", hint, strings.Join(names, ", "))
	}

	names := make([]string, 0, len(allowed))
	for _, file := range allowed {
		names = append(names, filepath.Base(file.Path))
	}
	sort.Strings(names)
	return nil, fmt.Errorf("no files match %q; available: %s", hint, strings.Join(names, ", "))
}

func (p *ModelScopeProvider) listModelFiles(ctx context.Context, modelID string) ([]ModelScopeFile, error) {
	url := fmt.Sprintf("%s/models/%s/files", modelscopeAPIURL, modelID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "LocalAIStack/1.0")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list model files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ModelScope API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var filesResp ModelScopeFilesResponse
	if err := json.Unmarshal(body, &filesResp); err != nil {
		return nil, fmt.Errorf("failed to parse files response: %w", err)
	}

	return filesResp.Data.Files, nil
}

func (p *ModelScopeProvider) downloadFile(ctx context.Context, url, destPath string, totalSize int64, progress func(downloaded, total int64)) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "LocalAIStack/1.0")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var downloaded int64
	buf := make([]byte, chunkSize)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := file.Write(buf[:n]); werr != nil {
				return werr
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, totalSize)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *ModelScopeProvider) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
	url := fmt.Sprintf("%s/models/%s", modelscopeAPIURL, modelID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "LocalAIStack/1.0")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get model info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("model %s not found", modelID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ModelScope API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var mm ModelScopeModel
	if err := json.Unmarshal(body, &mm); err != nil {
		return nil, fmt.Errorf("failed to parse model info: %w", err)
	}

	format := p.detectFormatFromTags(mm.Tags)

	return &ModelInfo{
		ID:          mm.ModelID,
		Name:        mm.Name,
		Description: mm.Description,
		Source:      SourceModelScope,
		Format:      format,
		Tags:        mm.Tags,
		Metadata: map[string]string{
			"downloads":  fmt.Sprintf("%d", mm.Downloads),
			"likes":      fmt.Sprintf("%d", mm.Likes),
			"visibility": mm.Visibility,
		},
	}, nil
}

func (p *ModelScopeProvider) Delete(ctx context.Context, modelID string) error {
	return fmt.Errorf("ModelScope models cannot be deleted via API")
}
