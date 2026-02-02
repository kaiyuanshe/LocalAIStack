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
	hfAPIURL          = "https://huggingface.co/api"
	hfModelURL        = "https://huggingface.co"
	hfAPITimeout      = 60 * time.Second
	hfDownloadTimeout = 30 * time.Minute
	chunkSize         = 1024 * 1024
)

type HuggingFaceProvider struct {
	client *http.Client
	token  string
}

func NewHuggingFaceProvider(token string) *HuggingFaceProvider {
	return &HuggingFaceProvider{
		client: &http.Client{Timeout: hfAPITimeout},
		token:  token,
	}
}

func (p *HuggingFaceProvider) Name() ModelSource {
	return SourceHuggingFace
}

type HFModel struct {
	ID           string   `json:"id"`
	ModelID      string   `json:"modelId"`
	Author       string   `json:"author"`
	Sha          string   `json:"sha"`
	LastModified string   `json:"lastModified"`
	Tags         []string `json:"tags"`
	Downloads    int      `json:"downloads"`
	Likes        int      `json:"likes"`
	Private      bool     `json:"private"`
	PipelineTag  string   `json:"pipeline_tag"`
	LibraryName  string   `json:"library_name"`
}

type HFModelFile struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func (p *HuggingFaceProvider) Search(ctx context.Context, query string, limit int) ([]ModelInfo, error) {
	if limit <= 0 {
		limit = 20
	}

	url := fmt.Sprintf("%s/models?search=%s&limit=%d&full=true", hfAPIURL, query, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search HuggingFace models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HuggingFace API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var hfModels []HFModel
	if err := json.Unmarshal(body, &hfModels); err != nil {
		return nil, fmt.Errorf("failed to parse HuggingFace response: %w", err)
	}

	var models []ModelInfo
	for _, hm := range hfModels {
		format := p.detectFormatFromTags(hm.Tags)

		models = append(models, ModelInfo{
			ID:          hm.ModelID,
			Name:        hm.ModelID,
			Description: fmt.Sprintf("Author: %s, Pipeline: %s", hm.Author, hm.PipelineTag),
			Source:      SourceHuggingFace,
			Format:      format,
			Tags:        hm.Tags,
			Metadata: map[string]string{
				"author":    hm.Author,
				"sha":       hm.Sha,
				"downloads": fmt.Sprintf("%d", hm.Downloads),
				"likes":     fmt.Sprintf("%d", hm.Likes),
				"library":   hm.LibraryName,
			},
		})
	}

	return models, nil
}

func (p *HuggingFaceProvider) detectFormatFromTags(tags []string) ModelFormat {
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

func (p *HuggingFaceProvider) Download(ctx context.Context, modelID string, destPath string, progress func(downloaded, total int64), opts DownloadOptions) error {
	files, err := p.listModelFiles(ctx, modelID)
	if err != nil {
		return fmt.Errorf("failed to list model files: %w", err)
	}

	modelDir := filepath.Join(destPath, strings.ReplaceAll(modelID, "/", "_"))
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	candidates, err := filterDownloadFiles(files, opts.FileHint)
	if err != nil {
		return err
	}

	for _, file := range candidates {
		fileURL := fmt.Sprintf("%s/%s/resolve/main/%s", hfModelURL, modelID, file.Path)
		destFile := filepath.Join(modelDir, filepath.Base(file.Path))

		if err := p.downloadFile(ctx, fileURL, destFile, file.Size, progress); err != nil {
			return fmt.Errorf("failed to download file %s: %w", file.Path, err)
		}
	}

	metadata := map[string]interface{}{
		"id":            modelID,
		"source":        "huggingface",
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

func filterDownloadFiles(files []HFModelFile, hint string) ([]HFModelFile, error) {
	allowed := make([]HFModelFile, 0, len(files))
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

	exact := make([]HFModelFile, 0)
	contains := make([]HFModelFile, 0)
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

func (p *HuggingFaceProvider) listModelFiles(ctx context.Context, modelID string) ([]HFModelFile, error) {
	url := fmt.Sprintf("%s/models/%s/tree/main", hfAPIURL, modelID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list model files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HuggingFace API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var files []HFModelFile
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, fmt.Errorf("failed to parse files response: %w", err)
	}

	return files, nil
}

func (p *HuggingFaceProvider) downloadFile(ctx context.Context, url, destPath string, totalSize int64, progress func(downloaded, total int64)) error {
	downloadClient := &http.Client{
		Timeout: hfDownloadTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := downloadClient.Do(req)
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

func (p *HuggingFaceProvider) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
	url := fmt.Sprintf("%s/models/%s", hfAPIURL, modelID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
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
		return nil, fmt.Errorf("HuggingFace API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var hm HFModel
	if err := json.Unmarshal(body, &hm); err != nil {
		return nil, fmt.Errorf("failed to parse model info: %w", err)
	}

	format := p.detectFormatFromTags(hm.Tags)

	return &ModelInfo{
		ID:          hm.ModelID,
		Name:        hm.ModelID,
		Description: fmt.Sprintf("Author: %s, Pipeline: %s", hm.Author, hm.PipelineTag),
		Source:      SourceHuggingFace,
		Format:      format,
		Tags:        hm.Tags,
		Metadata: map[string]string{
			"author":    hm.Author,
			"sha":       hm.Sha,
			"downloads": fmt.Sprintf("%d", hm.Downloads),
			"likes":     fmt.Sprintf("%d", hm.Likes),
			"library":   hm.LibraryName,
		},
	}, nil
}

func (p *HuggingFaceProvider) Delete(ctx context.Context, modelID string) error {
	return nil
}
