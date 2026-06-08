# Local Docs RAG Express

This recipe provides the Express RAG MVP for local documents.

Current MVP path:

- Index local Markdown, TXT, and PDF files into a local JSON index.
- Query the index with keyword retrieval.
- Optionally call an OpenAI-compatible generation endpoint for final answers.
- Return citations for every answer that has retrieved context.

Example:

```bash
./build/las express rag index ./docs
./build/las express rag serve --addr 127.0.0.1:18080
./build/las express rag healthcheck --base-url http://127.0.0.1:18080
./build/las express rag query "What is LocalAIStack?" --base-url http://127.0.0.1:18080
```

For a direct local query without starting the RAG API, omit `--base-url` and optionally pass `--llm-base-url`:

```bash
./build/las express rag query "What is LocalAIStack?" --llm-base-url http://127.0.0.1:8080
```

Qdrant and TEI are kept as recipe-level service targets for the next backend iteration.
