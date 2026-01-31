package modelmanager

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	ollamaAPIURL     = "https://ollama.com/api"
	ollamaLibraryURL = "https://ollama.com/library"
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

type OllamaLibraryModel struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Pulls       string   `json:"pulls"`
}

func (p *OllamaProvider) Search(ctx context.Context, query string, limit int) ([]ModelInfo, error) {
	if query == "" {
		return p.searchFromTags(ctx, query, limit)
	}

	libraryModels, err := p.searchFromLibrary(ctx, query)
	if err != nil {
		return p.searchFromTags(ctx, query, limit)
	}

	if len(libraryModels) == 0 {
		return p.searchFromTags(ctx, query, limit)
	}

	p.enrichModelsWithSizes(ctx, libraryModels, query)
	return libraryModels, nil
}

func (p *OllamaProvider) enrichModelsWithSizes(ctx context.Context, models []ModelInfo, query string) {
	tagModels, err := p.searchFromTags(ctx, query, 0)
	if err != nil || len(tagModels) == 0 {
		tagModels = nil
	}

	sizesByName := map[string]string{}
	tagsByName := map[string]string{}
	for _, model := range tagModels {
		if sizes, ok := model.Metadata["sizes"]; ok {
			sizesByName[model.Name] = sizes
		}
		if tags, ok := model.Metadata["tags"]; ok {
			tagsByName[model.Name] = tags
		}
	}

	for i := range models {
		if models[i].Metadata == nil {
			models[i].Metadata = map[string]string{}
		}
		if sizes, ok := sizesByName[models[i].Name]; ok && sizes != "" {
			models[i].Metadata["sizes"] = sizes
		} else {
			librarySizes, err := p.fetchLibrarySizes(ctx, models[i].Name)
			if err == nil && len(librarySizes) > 0 {
				models[i].Metadata["sizes"] = strings.Join(librarySizes, ", ")
			}
		}

		if tags, ok := tagsByName[models[i].Name]; ok && tags != "" {
			models[i].Metadata["tags"] = tags
		}
	}
}

func (p *OllamaProvider) fetchLibrarySizes(ctx context.Context, modelName string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/tags", ollamaLibraryURL, modelName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/html")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseLibraryTagSizes(string(body)), nil
}

func parseLibraryTagSizes(htmlContent string) []string {
	sizePattern := regexp.MustCompile(`x-test-size[^>]*>([^<]+)</span>`)
	matches := sizePattern.FindAllStringSubmatch(htmlContent, -1)
	if len(matches) == 0 {
		return nil
	}

	unique := map[string]struct{}{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		size := strings.TrimSpace(html.UnescapeString(match[1]))
		if size == "" {
			continue
		}
		unique[strings.ToLower(size)] = struct{}{}
	}

	sizes := make([]string, 0, len(unique))
	for size := range unique {
		sizes = append(sizes, size)
	}
	sort.Strings(sizes)
	return sizes
}

func tagSizeFromName(modelName string) string {
	parts := strings.SplitN(modelName, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	tag := strings.ToLower(strings.TrimSpace(parts[1]))
	if tag == "" {
		return ""
	}
	if ok, _ := regexp.MatchString(`^\d+(\.\d+)?b$`, tag); ok {
		return tag
	}
	return ""
}

func (p *OllamaProvider) searchFromLibrary(ctx context.Context, query string) ([]ModelInfo, error) {
	url := fmt.Sprintf("%s?q=%s", ollamaLibraryURL, query)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/html")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return p.parseLibraryHTML(string(body), query)
}

func (p *OllamaProvider) parseLibraryHTML(htmlContent string, query string) ([]ModelInfo, error) {
	var models []ModelInfo
	queryLower := strings.ToLower(query)

	modelPattern := regexp.MustCompile(`(?s)<div[^>]*x-test-model-title[^>]*title="([^"]+)"[^>]*>.*?<p[^>]*>([^<]+)</p>`)
	matches := modelPattern.FindAllStringSubmatch(htmlContent, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		name := strings.TrimSpace(html.UnescapeString(match[1]))
		description := strings.TrimSpace(html.UnescapeString(match[2]))

		if query != "" && !strings.Contains(strings.ToLower(name), queryLower) && !strings.Contains(strings.ToLower(description), queryLower) {
			continue
		}

		models = append(models, ModelInfo{
			ID:          name,
			Name:        name,
			Description: description,
			Source:      SourceOllama,
			Format:      FormatOllama,
			Tags:        []string{},
			Metadata:    map[string]string{},
		})
	}

	return models, nil
}

func (p *OllamaProvider) searchFromTags(ctx context.Context, query string, limit int) ([]ModelInfo, error) {
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
	seen := map[string]int{}
	sizeSets := map[string]map[string]struct{}{}
	tagSets := map[string]map[string]struct{}{}

	for _, om := range tagsResp.Models {
		if query != "" && !strings.Contains(strings.ToLower(om.Name), queryLower) {
			continue
		}

		baseName := om.Name
		hasTag := strings.Contains(om.Name, ":")
		if parts := strings.SplitN(om.Name, ":", 2); len(parts) > 0 {
			baseName = parts[0]
		}

		if _, ok := seen[baseName]; !ok {
			if limit > 0 && count >= limit {
				continue
			}
			seen[baseName] = len(models)
			models = append(models, ModelInfo{
				ID:          baseName,
				Name:        baseName,
				Description: "",
				Source:      SourceOllama,
				Format:      FormatOllama,
				Tags:        om.Details.Families,
				Metadata: map[string]string{
					"family": om.Details.Family,
				},
			})
			count++
		}

		size := strings.TrimSpace(om.Details.ParameterSize)
		if size == "" {
			size = tagSizeFromName(om.Name)
			if size == "" {
				size = FormatBytes(om.Size)
			}
		}
		if size != "" {
			if sizeSets[baseName] == nil {
				sizeSets[baseName] = map[string]struct{}{}
			}
			sizeSets[baseName][size] = struct{}{}
		}

		if tag := tagSizeFromName(om.Name); tag != "" {
			if tagSets[baseName] == nil {
				tagSets[baseName] = map[string]struct{}{}
			}
			tagSets[baseName][tag] = struct{}{}
		} else if parts := strings.SplitN(om.Name, ":", 2); len(parts) == 2 {
			tagName := strings.TrimSpace(parts[1])
			if tagName != "" {
				if tagSets[baseName] == nil {
					tagSets[baseName] = map[string]struct{}{}
				}
				tagSets[baseName][tagName] = struct{}{}
			}
		} else if !hasTag {
			if tagSets[baseName] == nil {
				tagSets[baseName] = map[string]struct{}{}
			}
			tagSets[baseName]["latest"] = struct{}{}
		}
	}

	for name, idx := range seen {
		sizes := make([]string, 0, len(sizeSets[name]))
		for size := range sizeSets[name] {
			sizes = append(sizes, size)
		}
		sort.Strings(sizes)

		tags := make([]string, 0, len(tagSets[name]))
		for tag := range tagSets[name] {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		if models[idx].Metadata == nil {
			models[idx].Metadata = map[string]string{}
		}
		models[idx].Metadata["sizes"] = strings.Join(sizes, ", ")
		if len(tags) > 0 {
			models[idx].Metadata["tags"] = strings.Join(tags, ", ")
		}
		family := models[idx].Metadata["family"]
		switch {
		case len(tags) > 0 && family != "":
			models[idx].Description = fmt.Sprintf("Tags: %s, Family: %s", strings.Join(tags, ", "), family)
		case len(tags) > 0:
			models[idx].Description = fmt.Sprintf("Tags: %s", strings.Join(tags, ", "))
		case family != "":
			models[idx].Description = fmt.Sprintf("Family: %s", family)
		}
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
