package rag

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const indexVersion = "0.1"

var supportedExt = map[string]bool{
	".md":       true,
	".markdown": true,
	".txt":      true,
	".pdf":      true,
}

func BuildIndex(sourceDir string, chunkSize, overlap int) (Index, error) {
	if chunkSize <= 0 {
		chunkSize = 800
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 4
	}

	var chunks []Chunk
	err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !supportedExt[ext] {
			return nil
		}
		text, err := readDocument(path)
		if err != nil {
			return fmt.Errorf("read document %s: %w", path, err)
		}
		text = normalizeWhitespace(text)
		if text == "" {
			return nil
		}
		title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		parts := splitChunks(text, chunkSize, overlap)
		for i, part := range parts {
			chunks = append(chunks, Chunk{
				ID:        chunkID(path, i, part),
				Text:      part,
				SourceURI: path,
				Title:     title,
				Metadata: map[string]any{
					"chunk_index": i,
					"extension":   ext,
				},
			})
		}
		return nil
	})
	if err != nil {
		return Index{}, err
	}
	sort.Slice(chunks, func(i, j int) bool {
		if chunks[i].SourceURI == chunks[j].SourceURI {
			return chunks[i].ID < chunks[j].ID
		}
		return chunks[i].SourceURI < chunks[j].SourceURI
	})
	return Index{Version: indexVersion, Chunks: chunks}, nil
}

func SaveIndex(path string, index Index) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func LoadIndex(path string) (Index, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Index{}, err
	}
	var index Index
	if err := json.Unmarshal(raw, &index); err != nil {
		return Index{}, err
	}
	return index, nil
}

func readDocument(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(filepath.Ext(path), ".pdf") {
		return extractPDFText(raw), nil
	}
	return string(raw), nil
}

func extractPDFText(raw []byte) string {
	var b strings.Builder
	for _, c := range raw {
		if c == '\n' || c == '\r' || c == '\t' || (c >= 32 && c <= 126) {
			b.WriteByte(c)
		} else {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

var whitespaceRE = regexp.MustCompile(`\s+`)

func normalizeWhitespace(text string) string {
	return strings.TrimSpace(whitespaceRE.ReplaceAllString(text, " "))
}

func splitChunks(text string, chunkSize, overlap int) []string {
	runes := []rune(text)
	if len(runes) <= chunkSize {
		return []string{text}
	}
	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}
	var chunks []string
	for start := 0; start < len(runes); start += step {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func chunkID(path string, index int, text string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:%d:%s", path, index, text)))
	return hex.EncodeToString(sum[:])[:16]
}
