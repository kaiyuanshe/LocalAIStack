package recipe

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Problems []string
}

func (e ValidationError) Error() string {
	return "recipe validation failed: " + strings.Join(e.Problems, "; ")
}

func Validate(r Recipe) error {
	var problems []string

	require := func(field, value string) {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, fmt.Sprintf("%s is required", field))
		}
	}

	require("id", r.ID)
	if strings.Contains(r.ID, " ") {
		problems = append(problems, "id must not contain spaces")
	}

	switch r.Kind {
	case KindInference:
		problems = append(problems, validateInference(r)...)
	case KindApplication:
		problems = append(problems, validateApplication(r)...)
	default:
		problems = append(problems, "kind must be one of: inference, application")
	}

	switch r.Tier {
	case TierExpress, TierExpert:
	default:
		problems = append(problems, "tier must be one of: express, expert")
	}

	switch r.Status {
	case StatusVerified, StatusExperimental, StatusDeprecated, StatusBroken:
	default:
		problems = append(problems, "status must be one of: verified, experimental, deprecated, broken")
	}

	if len(problems) > 0 {
		return ValidationError{Problems: problems}
	}
	return nil
}

func validateInference(r Recipe) []string {
	var problems []string
	if strings.TrimSpace(r.Model.ID) == "" {
		problems = append(problems, "model.id is required for inference recipes")
	}
	if strings.TrimSpace(r.Model.Format) == "" {
		problems = append(problems, "model.format is required for inference recipes")
	}
	if strings.TrimSpace(r.Runtime.API) == "" {
		problems = append(problems, "runtime.api is required for inference recipes")
	}
	if r.Runtime.Port < 0 || r.Runtime.Port > 65535 {
		problems = append(problems, "runtime.port must be between 0 and 65535")
	}
	if strings.TrimSpace(r.Validation.Healthcheck) == "" {
		problems = append(problems, "validation.healthcheck is required for inference recipes")
	}
	if strings.TrimSpace(r.Validation.Benchmark) == "" {
		problems = append(problems, "validation.benchmark is required for inference recipes")
	}
	if len(r.Engine) == 0 {
		problems = append(problems, "engine_config is required for inference recipes")
	}
	return problems
}

func validateApplication(r Recipe) []string {
	var problems []string
	if strings.TrimSpace(r.LLM.Recipe) == "" {
		problems = append(problems, "llm.recipe is required for application recipes")
	}
	if strings.TrimSpace(r.Embedding.Model) == "" {
		problems = append(problems, "embedding.model is required for application recipes")
	}
	if strings.TrimSpace(r.Retrieval.VectorStore) == "" {
		problems = append(problems, "retrieval.vector_store is required for application recipes")
	}
	if r.Retrieval.TopK < 0 {
		problems = append(problems, "retrieval.top_k must be non-negative")
	}
	if r.Retrieval.RerankTopK < 0 {
		problems = append(problems, "retrieval.rerank_top_k must be non-negative")
	}
	if r.Knowledge.Chunking.ChunkSize < 0 {
		problems = append(problems, "knowledge.chunking.chunk_size must be non-negative")
	}
	if r.Knowledge.Chunking.Overlap < 0 {
		problems = append(problems, "knowledge.chunking.overlap must be non-negative")
	}
	return problems
}
