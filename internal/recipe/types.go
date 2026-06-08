package recipe

type Kind string
type Tier string
type Status string

const (
	KindInference   Kind = "inference"
	KindApplication Kind = "application"

	TierExpress Tier = "express"
	TierExpert  Tier = "expert"

	StatusVerified     Status = "verified"
	StatusExperimental Status = "experimental"
	StatusDeprecated   Status = "deprecated"
	StatusBroken       Status = "broken"
)

type Recipe struct {
	ID          string              `yaml:"id"`
	Name        string              `yaml:"name,omitempty"`
	Description string              `yaml:"description,omitempty"`
	Kind        Kind                `yaml:"kind"`
	Tier        Tier                `yaml:"tier"`
	Status      Status              `yaml:"status"`
	Tags        []string            `yaml:"tags,omitempty"`
	Target      Target              `yaml:"target,omitempty"`
	Model       Model               `yaml:"model,omitempty"`
	Runtime     Runtime             `yaml:"runtime,omitempty"`
	Engine      EngineConfig        `yaml:"engine_config,omitempty"`
	Validation  Validation          `yaml:"validation,omitempty"`
	Fallbacks   map[string][]string `yaml:"fallbacks,omitempty"`
	LLM         LLMRef              `yaml:"llm,omitempty"`
	Embedding   Embedding           `yaml:"embedding,omitempty"`
	Knowledge   Knowledge           `yaml:"knowledge,omitempty"`
	Retrieval   Retrieval           `yaml:"retrieval,omitempty"`
	Answering   Answering           `yaml:"answering,omitempty"`
	Services    Services            `yaml:"services,omitempty"`
}

type Target struct {
	Hardware HardwareTarget `yaml:"hardware,omitempty"`
	OS       []string       `yaml:"os,omitempty"`
	Runtime  RuntimeTarget  `yaml:"runtime,omitempty"`
}

type HardwareTarget struct {
	Vendor         string   `yaml:"vendor,omitempty"`
	MinVRAMGB      int      `yaml:"min_vram_gb,omitempty"`
	MinRAMGB       int      `yaml:"min_ram_gb,omitempty"`
	RecommendedGPU []string `yaml:"recommended_gpu,omitempty"`
}

type RuntimeTarget struct {
	Docker bool   `yaml:"docker,omitempty"`
	Native bool   `yaml:"native,omitempty"`
	CUDA   string `yaml:"cuda,omitempty"`
	Metal  bool   `yaml:"metal,omitempty"`
}

type Model struct {
	Source       string `yaml:"source,omitempty"`
	ID           string `yaml:"id,omitempty"`
	Family       string `yaml:"family,omitempty"`
	Params       string `yaml:"params,omitempty"`
	Format       string `yaml:"format,omitempty"`
	Quantization string `yaml:"quantization,omitempty"`
}

type Runtime struct {
	Image string `yaml:"image,omitempty"`
	API   string `yaml:"api,omitempty"`
	Port  int    `yaml:"port,omitempty"`
	Mode  string `yaml:"mode,omitempty"`
}

type EngineConfig map[string]any

type Validation struct {
	Preflight   string `yaml:"preflight,omitempty"`
	Healthcheck string `yaml:"healthcheck,omitempty"`
	Benchmark   string `yaml:"benchmark,omitempty"`
}

type LLMRef struct {
	Recipe string `yaml:"recipe,omitempty"`
}

type Embedding struct {
	Provider string `yaml:"provider,omitempty"`
	Engine   string `yaml:"engine,omitempty"`
	Model    string `yaml:"model,omitempty"`
}

type Knowledge struct {
	Sources  []string       `yaml:"sources,omitempty"`
	Parsers  []string       `yaml:"parsers,omitempty"`
	Chunking ChunkingConfig `yaml:"chunking,omitempty"`
}

type ChunkingConfig struct {
	Strategy  string `yaml:"strategy,omitempty"`
	ChunkSize int    `yaml:"chunk_size,omitempty"`
	Overlap   int    `yaml:"overlap,omitempty"`
}

type Retrieval struct {
	VectorStore   string `yaml:"vector_store,omitempty"`
	KeywordSearch string `yaml:"keyword_search,omitempty"`
	HybridSearch  string `yaml:"hybrid_search,omitempty"`
	TopK          int    `yaml:"top_k,omitempty"`
	RerankTopK    int    `yaml:"rerank_top_k,omitempty"`
}

type Answering struct {
	Citations             string `yaml:"citations,omitempty"`
	FallbackWhenNoContext string `yaml:"fallback_when_no_context,omitempty"`
	AllowFreeformAnswer   *bool  `yaml:"allow_freeform_answer,omitempty"`
}

type Services struct {
	RAGAPI bool   `yaml:"rag_api,omitempty"`
	WebUI  string `yaml:"web_ui,omitempty"`
}
