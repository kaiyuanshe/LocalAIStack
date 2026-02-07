package modelmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultHFAPIURL   = "https://huggingface.co/api"
	defaultHFModelURL = "https://huggingface.co"
	hfAPITimeout      = 60 * time.Second
	hfDownloadTimeout = 30 * time.Minute
	chunkSize         = 1024 * 1024
)

type HuggingFaceProvider struct {
	client   *http.Client
	token    string
	apiURL   string
	modelURL string
}

func NewHuggingFaceProvider(token string) *HuggingFaceProvider {
	apiURL, modelURL := resolveHFEndpoints()
	return &HuggingFaceProvider{
		client:   &http.Client{Timeout: hfAPITimeout},
		token:    token,
		apiURL:   apiURL,
		modelURL: modelURL,
	}
}

func (p *HuggingFaceProvider) Name() ModelSource {
	return SourceHuggingFace
}

func resolveHFEndpoints() (string, string) {
	apiURL := defaultHFAPIURL
	modelURL := defaultHFModelURL

	if endpoint := strings.TrimSpace(os.Getenv("HF_ENDPOINT")); endpoint != "" {
		base := strings.TrimRight(endpoint, "/")
		apiURL = base + "/api"
		modelURL = base
	}
	if v := strings.TrimSpace(os.Getenv("HF_API_URL")); v != "" {
		apiURL = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("HF_MODEL_URL")); v != "" {
		modelURL = strings.TrimRight(v, "/")
	}
	return apiURL, modelURL
}

func shouldRetryHuggingFace(err error, statusCode int) bool {
	if statusCode == http.StatusTooManyRequests || statusCode >= 500 {
		return true
	}
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "tls handshake timeout") ||
		strings.Contains(msg, "unexpected eof") ||
		strings.Contains(msg, "eof")
}

type HFModel struct {
	ID           string      `json:"id"`
	ModelID      string      `json:"modelId"`
	Author       string      `json:"author"`
	Sha          string      `json:"sha"`
	LastModified string      `json:"lastModified"`
	Tags         []string    `json:"tags"`
	Downloads    int         `json:"downloads"`
	Likes        int         `json:"likes"`
	Private      bool        `json:"private"`
	PipelineTag  string      `json:"pipeline_tag"`
	LibraryName  string      `json:"library_name"`
	Siblings     []HFSibling `json:"siblings"`
}

type HFModelFile struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type HFSibling struct {
	RFilename string `json:"rfilename"`
	Size      int64  `json:"size"`
	LFS       *struct {
		Size int64 `json:"size"`
	} `json:"lfs"`
}

func (p *HuggingFaceProvider) Search(ctx context.Context, query string, limit int) ([]ModelInfo, error) {
	if limit <= 0 {
		limit = 20
	}

	url := fmt.Sprintf("%s/models?search=%s&limit=%d&full=true", p.apiURL, query, limit)

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
		fileURL := fmt.Sprintf("%s/%s/resolve/main/%s", p.modelURL, modelID, file.Path)
		relPath := filepath.Clean(filepath.FromSlash(file.Path))
		if relPath == "." || strings.HasPrefix(relPath, "..") {
			return fmt.Errorf("invalid file path %q", file.Path)
		}
		destFile := filepath.Join(modelDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destFile), 0755); err != nil {
			return fmt.Errorf("failed to create destination directory for %s: %w", file.Path, err)
		}

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
		allowed = append(allowed, file)
	}

	if hint == "" {
		sort.Slice(allowed, func(i, j int) bool {
			return allowed[i].Path < allowed[j].Path
		})
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
		pathLower := strings.ToLower(file.Path)
		if base == normalized || pathLower == normalized {
			exact = append(exact, file)
			continue
		}
		if strings.Contains(base, normalized) || strings.Contains(pathLower, normalized) {
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
			names = append(names, file.Path)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("multiple files match %q: %s", hint, strings.Join(names, ", "))
	}

	names := make([]string, 0, len(allowed))
	for _, file := range allowed {
		names = append(names, file.Path)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("no files match %q; available: %s", hint, strings.Join(names, ", "))
}

func (p *HuggingFaceProvider) listModelFiles(ctx context.Context, modelID string) ([]HFModelFile, error) {
	visited := map[string]struct{}{}
	queue := []string{""}
	allFiles := make([]HFModelFile, 0)

	for len(queue) > 0 {
		subPath := queue[0]
		queue = queue[1:]

		if _, ok := visited[subPath]; ok {
			continue
		}
		visited[subPath] = struct{}{}

		entries, err := p.listModelEntries(ctx, modelID, subPath)
		if err != nil {
			files, fallbackErr := p.listModelFilesFromSiblings(ctx, modelID)
			if fallbackErr != nil {
				return nil, err
			}
			return files, nil
		}

		for _, entry := range entries {
			switch strings.ToLower(entry.Type) {
			case "file":
				allFiles = append(allFiles, entry)
			case "directory", "dir", "tree":
				if strings.TrimSpace(entry.Path) != "" {
					queue = append(queue, entry.Path)
				}
			}
		}
	}

	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].Path < allFiles[j].Path
	})
	return allFiles, nil
}

func (p *HuggingFaceProvider) listModelFilesFromSiblings(ctx context.Context, modelID string) ([]HFModelFile, error) {
	url := fmt.Sprintf("%s/models/%s", p.apiURL, modelID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create fallback request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "LocalAIStack/1.0")
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request fallback model info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fallback model info returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read fallback response: %w", err)
	}

	var model HFModel
	if err := json.Unmarshal(body, &model); err != nil {
		return nil, fmt.Errorf("failed to parse fallback response: %w", err)
	}

	files := make([]HFModelFile, 0, len(model.Siblings))
	for _, s := range model.Siblings {
		path := strings.TrimSpace(s.RFilename)
		if path == "" {
			continue
		}
		size := s.Size
		if size <= 0 && s.LFS != nil {
			size = s.LFS.Size
		}
		files = append(files, HFModelFile{
			Type: "file",
			Path: path,
			Size: size,
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func (p *HuggingFaceProvider) listModelEntries(ctx context.Context, modelID, subPath string) ([]HFModelFile, error) {
	apiURL := fmt.Sprintf("%s/models/%s/tree/main", p.apiURL, modelID)
	if strings.TrimSpace(subPath) != "" {
		apiURL = fmt.Sprintf("%s/%s", apiURL, encodeHFPath(subPath))
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
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
			lastErr = err
			if attempt < 3 && shouldRetryHuggingFace(err, 0) {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("failed to list model files: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < 3 && shouldRetryHuggingFace(readErr, resp.StatusCode) {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("failed to read response: %w", readErr)
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HuggingFace API returned status %d", resp.StatusCode)
			if attempt < 3 && shouldRetryHuggingFace(nil, resp.StatusCode) {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			return nil, lastErr
		}

		var files []HFModelFile
		if err := json.Unmarshal(body, &files); err != nil {
			return nil, fmt.Errorf("failed to parse files response: %w", err)
		}
		return files, nil
	}
	return nil, fmt.Errorf("failed to list model files: %w", lastErr)
}

func encodeHFPath(path string) string {
	parts := strings.Split(path, "/")
	encoded := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		encoded = append(encoded, url.PathEscape(part))
	}
	return strings.Join(encoded, "/")
}

func (p *HuggingFaceProvider) DownloadSupportFiles(ctx context.Context, modelID string, destPath string) error {
	files, err := p.listModelFiles(ctx, modelID)
	if err != nil {
		return fmt.Errorf("failed to list model files: %w", err)
	}
	modelDir := filepath.Join(destPath, strings.ReplaceAll(modelID, "/", "_"))
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	remote := map[string]HFModelFile{}
	for _, file := range files {
		if file.Type != "file" {
			continue
		}
		base := strings.ToLower(filepath.Base(file.Path))
		if IsRequiredModelFile(base) {
			remote[base] = file
		}
	}

	for _, base := range RequiredModelFiles() {
		destFile := filepath.Join(modelDir, base)
		if _, err := os.Stat(destFile); err == nil {
			continue
		}
		remoteFile, ok := remote[base]
		if !ok {
			continue
		}
		fileURL := fmt.Sprintf("%s/%s/resolve/main/%s", p.modelURL, modelID, remoteFile.Path)
		if err := p.downloadFile(ctx, fileURL, destFile, remoteFile.Size, nil); err != nil {
			return fmt.Errorf("failed to download file %s: %w", remoteFile.Path, err)
		}
	}

	return nil
}

func (p *HuggingFaceProvider) downloadFile(ctx context.Context, url, destPath string, totalSize int64, progress func(downloaded, total int64)) error {
	downloadClient := &http.Client{
		Timeout: hfDownloadTimeout,
	}
	if p.client != nil {
		downloadClient.Transport = p.client.Transport
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		file, err := os.Create(destPath)
		if err != nil {
			return err
		}

		resp, err := downloadClient.Do(req)
		if err != nil {
			_ = file.Close()
			lastErr = err
			if attempt < 3 && shouldRetryHuggingFace(err, 0) {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			return err
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			_ = file.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			if attempt < 3 && shouldRetryHuggingFace(nil, resp.StatusCode) {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			return lastErr
		}

		var downloaded int64
		buf := make([]byte, chunkSize)
		readOK := true
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				if _, werr := file.Write(buf[:n]); werr != nil {
					_ = resp.Body.Close()
					_ = file.Close()
					return werr
				}
				downloaded += int64(n)
				if progress != nil {
					progress(downloaded, totalSize)
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				lastErr = readErr
				readOK = false
				break
			}
		}
		_ = resp.Body.Close()
		_ = file.Close()

		if readOK {
			return nil
		}
		if attempt < 3 && shouldRetryHuggingFace(lastErr, 0) {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		return lastErr
	}
	return lastErr
}

func (p *HuggingFaceProvider) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
	url := fmt.Sprintf("%s/models/%s", p.apiURL, modelID)

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
