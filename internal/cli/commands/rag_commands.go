package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/rag"
)

func newExpressRAGCommand() *cobra.Command {
	var indexPath string
	ragCmd := &cobra.Command{
		Use:   "rag",
		Short: "Run Express local-docs RAG workflows",
	}
	ragCmd.PersistentFlags().StringVar(&indexPath, "index", defaultRAGIndexPath(), "RAG index path")

	indexCmd := &cobra.Command{
		Use:   "index [source-dir]",
		Short: "Index local Markdown, TXT, and PDF documents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			chunkSize, _ := cmd.Flags().GetInt("chunk-size")
			overlap, _ := cmd.Flags().GetInt("overlap")
			index, err := rag.BuildIndex(args[0], chunkSize, overlap)
			if err != nil {
				return err
			}
			if err := rag.SaveIndex(indexPath, index); err != nil {
				return err
			}
			cmd.Printf("Indexed %d chunks into %s\n", len(index.Chunks), indexPath)
			return nil
		},
	}
	indexCmd.Flags().Int("chunk-size", 800, "Chunk size in runes")
	indexCmd.Flags().Int("overlap", 120, "Chunk overlap in runes")

	queryCmd := &cobra.Command{
		Use:   "query [question]",
		Short: "Query the RAG API or local RAG index",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, _ := cmd.Flags().GetString("base-url")
			llmBaseURL, _ := cmd.Flags().GetString("llm-base-url")
			model, _ := cmd.Flags().GetString("model")
			topK, _ := cmd.Flags().GetInt("top-k")
			if strings.TrimSpace(baseURL) != "" {
				body, err := queryRAGAPI(baseURL, args[0], topK, model)
				if err != nil {
					return err
				}
				cmd.Println(strings.TrimSpace(string(body)))
				return nil
			}
			index, err := rag.LoadIndex(indexPath)
			if err != nil {
				return err
			}
			resp, err := rag.Query(cmd.Context(), index, args[0], rag.QueryOptions{
				TopK:    topK,
				BaseURL: llmBaseURL,
				Model:   model,
			})
			if err != nil {
				return err
			}
			printRAGResponse(cmd, resp)
			return nil
		},
	}
	queryCmd.Flags().String("base-url", "", "RAG API base URL")
	queryCmd.Flags().String("llm-base-url", "", "OpenAI-compatible generation base URL for direct local query")
	queryCmd.Flags().String("model", "", "Model name for generation")
	queryCmd.Flags().Int("top-k", 5, "Number of chunks to retrieve")

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the local RAG API",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, _ := cmd.Flags().GetString("addr")
			baseURL, _ := cmd.Flags().GetString("base-url")
			model, _ := cmd.Flags().GetString("model")
			topK, _ := cmd.Flags().GetInt("top-k")
			index, err := rag.LoadIndex(indexPath)
			if err != nil {
				return err
			}
			server := rag.NewServer(rag.ServerOptions{
				Addr:    addr,
				Index:   index,
				BaseURL: baseURL,
				Model:   model,
				TopK:    topK,
			})
			cmd.Printf("Serving RAG API on http://%s\n", server.Addr)
			return server.ListenAndServe()
		},
	}
	serveCmd.Flags().String("addr", "127.0.0.1:18080", "RAG API listen address")
	serveCmd.Flags().String("base-url", "", "OpenAI-compatible generation base URL")
	serveCmd.Flags().String("model", "", "Model name for generation")
	serveCmd.Flags().Int("top-k", 5, "Number of chunks to retrieve")

	healthCmd := &cobra.Command{
		Use:   "healthcheck",
		Short: "Check the RAG API health endpoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, _ := cmd.Flags().GetString("base-url")
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get(strings.TrimRight(baseURL, "/") + "/health")
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("RAG healthcheck returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			}
			cmd.Printf("RAG healthcheck OK: %s\n", strings.TrimSpace(string(body)))
			return nil
		},
	}
	healthCmd.Flags().String("base-url", "http://127.0.0.1:18080", "RAG API base URL")

	ragCmd.AddCommand(indexCmd, queryCmd, serveCmd, healthCmd)
	return ragCmd
}

func defaultRAGIndexPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".localaistack", "rag", "local-docs", "index.json")
	}
	return filepath.Join(home, ".localaistack", "rag", "local-docs", "index.json")
}

func printRAGResponse(cmd *cobra.Command, resp rag.QueryResponse) {
	cmd.Printf("Answer:\n%s\n", resp.Answer)
	if len(resp.Citations) == 0 {
		cmd.Println("Citations: none")
		return
	}
	cmd.Println("Citations:")
	for i, citation := range resp.Citations {
		cmd.Printf("[%d] %s score=%.3f chunk=%s\n", i+1, citation.SourceURI, citation.Score, citation.ChunkID)
		cmd.Printf("    %s\n", citation.Snippet)
	}
}

func queryRAGAPI(baseURL, question string, topK int, model string) ([]byte, error) {
	payloadMap := map[string]any{"query": question}
	if topK > 0 {
		payloadMap["top_k"] = topK
	}
	if strings.TrimSpace(model) != "" {
		payloadMap["model"] = model
	}
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(strings.TrimRight(baseURL, "/")+"/v1/rag/query", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("RAG query returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
