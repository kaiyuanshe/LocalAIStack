# LocalAIStack Recipes

This directory contains LocalAIStack 2.0 recipes.

Recipe kinds:

- `inference`: describes how a generation model runs on target hardware.
- `application`: composes inference, embedding, indexing, retrieval, answering, and services.

Each concrete recipe should live in its own directory and use `recipe.yaml`.

Validate recipes:

```bash
go run ./cmd/cli recipes validate
go run ./cmd/cli recipes list
```

