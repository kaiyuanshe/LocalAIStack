package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

var tokenRE = regexp.MustCompile(`[[:alnum:]_]+`)

func Query(ctx context.Context, index Index, question string, opts QueryOptions) (QueryResponse, error) {
	topK := opts.TopK
	if topK <= 0 {
		topK = 5
	}
	citations := retrieve(index, question, topK)
	if len(citations) == 0 {
		return QueryResponse{
			Answer:    "I could not find supporting context in the indexed documents.",
			Citations: nil,
			Found:     false,
		}, nil
	}

	answer := answerFromContext(question, citations)
	if strings.TrimSpace(opts.BaseURL) != "" {
		generated, err := generateAnswer(ctx, opts.BaseURL, opts.Model, question, citations)
		if err == nil && strings.TrimSpace(generated) != "" {
			answer = generated
		}
	}
	return QueryResponse{Answer: answer, Citations: citations, Found: true}, nil
}

func retrieve(index Index, question string, topK int) []Citation {
	queryTokens := tokenSet(question)
	if len(queryTokens) == 0 {
		return nil
	}
	type scored struct {
		chunk Chunk
		score float64
	}
	var scoredChunks []scored
	for _, chunk := range index.Chunks {
		chunkTokens := tokenSet(chunk.Text)
		if len(chunkTokens) == 0 {
			continue
		}
		score := cosine(queryTokens, chunkTokens)
		if score <= 0 {
			continue
		}
		scoredChunks = append(scoredChunks, scored{chunk: chunk, score: score})
	}
	sort.Slice(scoredChunks, func(i, j int) bool {
		if scoredChunks[i].score == scoredChunks[j].score {
			return scoredChunks[i].chunk.ID < scoredChunks[j].chunk.ID
		}
		return scoredChunks[i].score > scoredChunks[j].score
	})
	if len(scoredChunks) > topK {
		scoredChunks = scoredChunks[:topK]
	}
	citations := make([]Citation, 0, len(scoredChunks))
	for _, item := range scoredChunks {
		citations = append(citations, Citation{
			ChunkID:   item.chunk.ID,
			SourceURI: item.chunk.SourceURI,
			Title:     item.chunk.Title,
			Score:     item.score,
			Snippet:   snippet(item.chunk.Text, 360),
		})
	}
	return citations
}

func tokenSet(text string) map[string]float64 {
	tokens := tokenRE.FindAllString(strings.ToLower(text), -1)
	result := make(map[string]float64, len(tokens))
	for _, token := range tokens {
		if len(token) < 2 {
			continue
		}
		result[token]++
	}
	return result
}

func cosine(a, b map[string]float64) float64 {
	var dot, normA, normB float64
	for _, value := range a {
		normA += value * value
	}
	for _, value := range b {
		normB += value * value
	}
	for token, value := range a {
		dot += value * b[token]
	}
	if dot == 0 || normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func answerFromContext(question string, citations []Citation) string {
	if len(citations) == 0 {
		return "I could not find supporting context in the indexed documents."
	}
	return fmt.Sprintf("I found relevant context for %q in the indexed documents. Use the citations to inspect the source text.", question)
}

func generateAnswer(ctx context.Context, baseURL, model, question string, citations []Citation) (string, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if model == "" {
		model = "default"
	}
	var contextBuilder strings.Builder
	for i, citation := range citations {
		contextBuilder.WriteString(fmt.Sprintf("[%d] %s\n%s\n\n", i+1, citation.SourceURI, citation.Snippet))
	}
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "Answer only from the provided context. If the context is insufficient, say you could not find supporting context. Include citation markers like [1]."},
			{"role": "user", "content": fmt.Sprintf("Context:\n%s\nQuestion: %s", contextBuilder.String(), question)},
		},
		"temperature": 0,
		"max_tokens":  512,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("chat completion returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	if len(decoded.Choices) == 0 {
		return "", fmt.Errorf("chat completion response has no choices")
	}
	return strings.TrimSpace(decoded.Choices[0].Message.Content), nil
}

func snippet(text string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}
