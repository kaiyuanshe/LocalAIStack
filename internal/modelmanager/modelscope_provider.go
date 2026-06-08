package modelmanager

import (
	"bytes"
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
	"unicode"
)

const (
	defaultModelScopeEndpoint = "https://modelscope.cn"
	modelscopeTimeout         = 60 * time.Second
)

type ModelScopeProvider struct {
	client   *http.Client
	token    string
	endpoint string
}

func NewModelScopeProvider(token string) *ModelScopeProvider {
	return &ModelScopeProvider{
		client:   &http.Client{Timeout: modelscopeTimeout},
		token:    token,
		endpoint: resolveModelScopeEndpoint(),
	}
}

func (p *ModelScopeProvider) Name() ModelSource {
	return SourceModelScope
}

type ModelScopeModel struct {
	ModelID     string                     `json:"ModelId"`
	Backend     *ModelScopeBackend         `json:"BackendSupport"`
	Path        string                     `json:"Path"`
	Name        string                     `json:"Name"`
	ChineseName string                     `json:"ChineseName"`
	Description string                     `json:"Description"`
	Tags        []string                   `json:"Tags"`
	Libraries   []string                   `json:"Libraries"`
	Frameworks  []string                   `json:"Frameworks"`
	ModelInfos  map[string]json.RawMessage `json:"ModelInfos"`
	Downloads   int                        `json:"Downloads"`
	Likes       int                        `json:"Likes"`
	Stars       int                        `json:"Stars"`
	StorageSize int64                      `json:"StorageSize"`
	Visibility  any                        `json:"Visibility"`
}

type ModelScopeFile struct {
	Path string `json:"Path"`
	Size int64  `json:"Size"`
	Type string `json:"Type"`
}

type ModelScopeSearchResponse struct {
	Code int `json:"Code"`
	Data struct {
		Models []ModelScopeModel `json:"Models"`
	} `json:"Data"`
	Message string `json:"Message"`
	Success bool   `json:"Success"`
}

type ModelScopeFilesResponse struct {
	Data struct {
		Files []ModelScopeFile `json:"Files"`
	} `json:"Data"`
}

type ModelScopeInfoResponse struct {
	Code    int             `json:"Code"`
	Data    ModelScopeModel `json:"Data"`
	Message string          `json:"Message"`
	Success bool            `json:"Success"`
}

type ModelScopeBackend struct {
	ModelID string `json:"model_id"`
}

func resolveModelScopeEndpoint() string {
	if endpoint := strings.TrimSpace(os.Getenv("MODELSCOPE_ENDPOINT")); endpoint != "" {
		return strings.TrimRight(endpoint, "/")
	}
	if endpoint := strings.TrimSpace(os.Getenv("MODELSCOPE_DOMAIN")); endpoint != "" {
		endpoint = strings.TrimRight(endpoint, "/")
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = "https://" + endpoint
		}
		return endpoint
	}
	return defaultModelScopeEndpoint
}

func (p *ModelScopeProvider) Search(ctx context.Context, query string, limit int) ([]ModelInfo, error) {
	if limit <= 0 {
		limit = 20
	}

	models, err := p.searchModelScopeByName(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	if len(models) > 0 {
		return models, nil
	}

	models, err = p.searchModelScopePath(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	if len(models) > 0 {
		return models, nil
	}

	for _, candidate := range modelScopeSearchFallbackPaths(query) {
		models, err := p.searchModelScopeByName(ctx, candidate, limit*4)
		if err != nil {
			continue
		}
		models = filterModelScopeSearchResults(models, query, limit)
		if len(models) > 0 {
			return models, nil
		}

		models, err = p.searchModelScopePath(ctx, candidate, limit*4)
		if err != nil {
			continue
		}
		models = filterModelScopeSearchResults(models, query, limit)
		if len(models) > 0 {
			return models, nil
		}
	}

	return models, nil
}

func (p *ModelScopeProvider) searchModelScopeByName(ctx context.Context, query string, limit int) ([]ModelInfo, error) {
	return p.searchModelScope(ctx, fmt.Sprintf(`{"Name":%q,"PageNumber":1,"PageSize":%d}`, query, limit))
}

func (p *ModelScopeProvider) searchModelScopePath(ctx context.Context, query string, limit int) ([]ModelInfo, error) {
	return p.searchModelScope(ctx, fmt.Sprintf(`{"Path":%q,"PageNumber":1,"PageSize":%d}`, query, limit))
}

func (p *ModelScopeProvider) searchModelScope(ctx context.Context, payload string) ([]ModelInfo, error) {
	url := fmt.Sprintf("%s/api/v1/models/", strings.TrimRight(p.endpoint, "/"))

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBufferString(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
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
	if searchResp.Code != 0 && searchResp.Code != http.StatusOK {
		return nil, fmt.Errorf("ModelScope API returned code %d: %s", searchResp.Code, searchResp.Message)
	}

	var models []ModelInfo
	for _, mm := range searchResp.Data.Models {
		models = append(models, p.toModelInfo(mm))
	}

	return models, nil
}

func modelScopeSearchFallbackPaths(query string) []string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return nil
	}

	withoutTrailingVersion := strings.TrimRightFunc(trimmed, func(r rune) bool {
		return unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' || unicode.IsSpace(r)
	})
	withoutTrailingVersion = strings.TrimRight(withoutTrailingVersion, ".-_ ")
	if withoutTrailingVersion == "" || strings.EqualFold(withoutTrailingVersion, trimmed) {
		return nil
	}
	return []string{withoutTrailingVersion}
}

func filterModelScopeSearchResults(models []ModelInfo, query string, limit int) []ModelInfo {
	normalized := strings.ToLower(strings.TrimSpace(query))
	if normalized == "" {
		return models
	}

	filtered := make([]ModelInfo, 0, len(models))
	for _, model := range models {
		haystack := strings.ToLower(model.ID + " " + model.Name + " " + model.Description + " " + strings.Join(model.Tags, " "))
		if strings.Contains(haystack, normalized) {
			filtered = append(filtered, model)
			if limit > 0 && len(filtered) >= limit {
				break
			}
		}
	}
	return filtered
}

func (p *ModelScopeProvider) toModelInfo(mm ModelScopeModel) ModelInfo {
	id := strings.TrimSpace(mm.ModelID)
	if id == "" && mm.Backend != nil {
		id = strings.TrimSpace(mm.Backend.ModelID)
	}
	if id == "" && strings.TrimSpace(mm.Path) != "" && strings.TrimSpace(mm.Name) != "" {
		id = strings.Trim(strings.TrimSpace(mm.Path), "/") + "/" + strings.Trim(strings.TrimSpace(mm.Name), "/")
	}
	if id == "" {
		id = strings.TrimSpace(mm.Name)
	}

	name := strings.TrimSpace(id)
	if name == "" {
		name = strings.TrimSpace(mm.Name)
	}
	description := strings.TrimSpace(mm.Description)
	if description == "" {
		description = strings.TrimSpace(mm.ChineseName)
	}

	metadata := map[string]string{
		"downloads": fmt.Sprintf("%d", mm.Downloads),
		"stars":     fmt.Sprintf("%d", mm.Stars),
	}
	if mm.Likes > 0 {
		metadata["likes"] = fmt.Sprintf("%d", mm.Likes)
	}
	if mm.Visibility != nil {
		metadata["visibility"] = fmt.Sprint(mm.Visibility)
	}
	if mm.StorageSize > 0 {
		metadata["storage_size"] = fmt.Sprintf("%d", mm.StorageSize)
	}
	if len(mm.Libraries) > 0 {
		metadata["libraries"] = strings.Join(mm.Libraries, ",")
	}

	return ModelInfo{
		ID:          id,
		Name:        name,
		Description: description,
		Source:      SourceModelScope,
		Format:      p.detectFormat(mm),
		Size:        mm.StorageSize,
		Tags:        mm.Tags,
		Metadata:    metadata,
	}
}

func (p *ModelScopeProvider) detectFormat(mm ModelScopeModel) ModelFormat {
	for key := range mm.ModelInfos {
		keyLower := strings.ToLower(key)
		if strings.Contains(keyLower, "gguf") {
			return FormatGGUF
		}
		if strings.Contains(keyLower, "safetensor") {
			return FormatSafetensors
		}
	}
	for _, value := range append(append([]string{}, mm.Tags...), append(mm.Libraries, mm.Frameworks...)...) {
		tagLower := strings.ToLower(value)
		if strings.Contains(tagLower, "gguf") {
			return FormatGGUF
		}
		if strings.Contains(tagLower, "safetensor") {
			return FormatSafetensors
		}
	}
	return FormatUnknown
}

func (p *ModelScopeProvider) Download(ctx context.Context, modelID string, destPath string, progress func(downloaded, total int64), opts DownloadOptions) error {
	return downloadModelWithModelScopeCLI(ctx, destPath, modelID, opts)
}

func filterModelScopeFiles(files []ModelScopeFile, hint string) ([]ModelScopeFile, error) {
	allowed := make([]ModelScopeFile, 0, len(files))
	required := make([]ModelScopeFile, 0)
	for _, file := range files {
		if file.Type != "file" {
			continue
		}
		base := strings.ToLower(filepath.Base(file.Path))
		if IsRequiredModelFile(base) {
			required = append(required, file)
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Path))
		if ext != ".gguf" && ext != ".safetensors" && ext != ".bin" {
			continue
		}
		allowed = append(allowed, file)
	}

	if hint == "" {
		return append(allowed, required...), nil
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
		return append(exact, required...), nil
	}
	if len(exact) > 1 {
		return nil, fmt.Errorf("multiple files match %q; please specify a more specific filename", hint)
	}
	if len(contains) == 1 {
		return append(contains, required...), nil
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
	url := fmt.Sprintf("%s/api/v1/models/%s/files", strings.TrimRight(p.endpoint, "/"), modelID)

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

func (p *ModelScopeProvider) DownloadSupportFiles(ctx context.Context, modelID string, destPath string) error {
	files, err := p.listModelFiles(ctx, modelID)
	if err != nil {
		return fmt.Errorf("failed to list model files: %w", err)
	}
	modelDir := filepath.Join(destPath, strings.ReplaceAll(modelID, "/", "_"))
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	remote := map[string]ModelScopeFile{}
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
		fileURL := fmt.Sprintf("%s/api/v1/models/%s/repo?file_path=%s", strings.TrimRight(p.endpoint, "/"), modelID, remoteFile.Path)
		if err := p.downloadFile(ctx, fileURL, destFile, remoteFile.Size, nil); err != nil {
			return fmt.Errorf("failed to download file %s: %w", remoteFile.Path, err)
		}
	}

	return nil
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
	url := fmt.Sprintf("%s/api/v1/models/%s", strings.TrimRight(p.endpoint, "/"), modelID)

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
	var infoResp ModelScopeInfoResponse
	if err := json.Unmarshal(body, &infoResp); err == nil && (infoResp.Code == 0 || infoResp.Code == http.StatusOK) {
		mm = infoResp.Data
	} else if err := json.Unmarshal(body, &mm); err != nil {
		return nil, fmt.Errorf("failed to parse model info: %w", err)
	}

	info := p.toModelInfo(mm)
	return &info, nil
}

func (p *ModelScopeProvider) Delete(ctx context.Context, modelID string) error {
	return fmt.Errorf("ModelScope models cannot be deleted via API")
}
