package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/express"
	"github.com/zhuangbiaowei/LocalAIStack/internal/recipe"
)

func RegisterExpressCommands(rootCmd *cobra.Command) {
	var recipesRoot string
	expressCmd := &cobra.Command{
		Use:   "express",
		Short: "Run verified Express recipes",
	}
	expressCmd.PersistentFlags().StringVar(&recipesRoot, "recipes-root", "", "recipes root directory")

	detectCmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect hardware and runtime facts",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			normalized, _ := cmd.Flags().GetBool("normalized")
			baseInfoPath, err := express.RefreshBaseInfo()
			if err != nil {
				return err
			}
			if output == "json" && !normalized {
				raw, err := express.ReadBaseInfoRaw(baseInfoPath)
				if err != nil {
					return err
				}
				cmd.Print(string(raw))
				if !strings.HasSuffix(string(raw), "\n") {
					cmd.Println()
				}
				return nil
			}
			facts, err := express.LoadHardwareFacts(baseInfoPath)
			if err != nil {
				return err
			}
			if output == "json" && normalized {
				raw, err := facts.JSON()
				if err != nil {
					return err
				}
				cmd.Println(string(raw))
				return nil
			}
			cmd.Println(facts.Summary())
			return nil
		},
	}
	detectCmd.Flags().String("output", "text", "Output format: text or json")
	detectCmd.Flags().Bool("normalized", false, "Print selector-normalized facts when output is json")

	recommendCmd := &cobra.Command{
		Use:   "recommend",
		Short: "Recommend Express inference recipes for this machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			workload, _ := cmd.Flags().GetString("workload")
			modelSize, _ := cmd.Flags().GetString("model-size")
			memoryMode, _ := cmd.Flags().GetString("memory-mode")
			limit, _ := cmd.Flags().GetInt("limit")
			recordsRoot, err := resolveRecipesRoot(recipesRoot)
			if err != nil {
				return err
			}
			registry, err := recipe.LoadRegistryFromDir(recordsRoot)
			if err != nil {
				return err
			}
			facts, err := express.LoadHardwareFacts(express.DefaultBaseInfoPath())
			if err != nil {
				return fmt.Errorf("failed to read base info at %s (run `./build/las init` or `./build/las express detect`): %w", express.DefaultBaseInfoPath(), err)
			}
			recs := express.RecommendRecipes(registry, facts, express.Intent{
				Workload:   workload,
				ModelSize:  modelSize,
				MemoryMode: memoryMode,
			})
			if limit > 0 && len(recs) > limit {
				recs = recs[:limit]
			}
			if output == "json" {
				raw, err := json.MarshalIndent(recs, "", "  ")
				if err != nil {
					return err
				}
				cmd.Println(string(raw))
				return nil
			}
			printRecommendations(cmd, recs)
			return nil
		},
	}
	recommendCmd.Flags().String("workload", "api", "Target workload: api, chat, rag, agent")
	recommendCmd.Flags().String("model-size", "", "Preferred model size such as 8b, 14b, 32b")
	recommendCmd.Flags().String("memory-mode", "balanced", "Memory strategy: safe, balanced, aggressive")
	recommendCmd.Flags().Int("limit", 5, "Maximum recommendations to print")
	recommendCmd.Flags().String("output", "text", "Output format: text or json")

	configCmd := &cobra.Command{
		Use:   "config [recipe-id]",
		Short: "Print generated runtime config hints for a recipe",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			record, err := loadRecipeByID(recipesRoot, args[0])
			if err != nil {
				return err
			}
			printConfigHints(cmd, record.Recipe)
			return nil
		},
	}

	preflightCmd := &cobra.Command{
		Use:   "preflight [recipe-id]",
		Short: "Check whether an inference recipe can run on this machine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			record, err := loadRecipeByID(recipesRoot, args[0])
			if err != nil {
				return err
			}
			report := express.Preflight(record)
			printPreflightReport(cmd, report)
			if report.Failed() {
				return fmt.Errorf("preflight failed for %s", report.RecipeID)
			}
			return nil
		},
	}

	runCmd := &cobra.Command{
		Use:   "run [recipe-id]",
		Short: "Run an Express inference recipe",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			skipPreflight, _ := cmd.Flags().GetBool("skip-preflight")
			record, err := loadRecipeByID(recipesRoot, args[0])
			if err != nil {
				return err
			}
			if !skipPreflight {
				report := express.Preflight(record)
				if report.Failed() {
					printPreflightReport(cmd, report)
					return fmt.Errorf("preflight failed for %s", report.RecipeID)
				}
			}
			plan := express.BuildRunPlan(record)
			printRunPlan(cmd, plan)
			if dryRun {
				return nil
			}
			_, err = express.Run(record)
			return err
		},
	}
	runCmd.Flags().Bool("dry-run", false, "Print the run command without starting it")
	runCmd.Flags().Bool("skip-preflight", false, "Skip preflight checks before running")

	healthcheckCmd := &cobra.Command{
		Use:   "healthcheck [recipe-id]",
		Short: "Check an OpenAI-compatible generation endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			record, err := loadRecipeByID(recipesRoot, args[0])
			if err != nil {
				return err
			}
			baseURL, _ := cmd.Flags().GetString("base-url")
			model, _ := cmd.Flags().GetString("model")
			timeout, _ := cmd.Flags().GetDuration("timeout")
			if baseURL == "" {
				baseURL = baseURLForRecipe(record.Recipe)
			}
			report, err := express.Healthcheck(cmd.Context(), express.HealthOptions{
				BaseURL: baseURL,
				Model:   model,
				Timeout: timeout,
			})
			if err != nil {
				return err
			}
			cmd.Printf("Healthcheck OK: base_url=%s models=%d latency=%s preview=%q\n",
				report.BaseURL, report.ModelCount, report.Latency.Round(time.Millisecond), report.ResponsePreview)
			return nil
		},
	}
	healthcheckCmd.Flags().String("base-url", "", "OpenAI-compatible base URL")
	healthcheckCmd.Flags().String("model", "", "Model name to use for chat completion")
	healthcheckCmd.Flags().Duration("timeout", 30*time.Second, "HTTP timeout")

	benchmarkCmd := &cobra.Command{
		Use:   "benchmark [recipe-id]",
		Short: "Run a lightweight OpenAI-compatible benchmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			record, err := loadRecipeByID(recipesRoot, args[0])
			if err != nil {
				return err
			}
			baseURL, _ := cmd.Flags().GetString("base-url")
			model, _ := cmd.Flags().GetString("model")
			requests, _ := cmd.Flags().GetInt("requests")
			timeout, _ := cmd.Flags().GetDuration("timeout")
			if baseURL == "" {
				baseURL = baseURLForRecipe(record.Recipe)
			}
			report, err := express.Benchmark(cmd.Context(), express.BenchmarkOptions{
				BaseURL:  baseURL,
				Model:    model,
				Requests: requests,
				Timeout:  timeout,
			})
			if err != nil {
				return err
			}
			cmd.Printf("Benchmark: base_url=%s requests=%d success=%d failed=%d avg_latency=%s approx_tokens_per_sec=%.2f\n",
				report.BaseURL,
				report.Requests,
				report.Successful,
				report.Failed,
				report.AverageLatency.Round(time.Millisecond),
				report.ApproxTokensSec,
			)
			if report.Successful == 0 {
				return fmt.Errorf("benchmark completed with no successful requests")
			}
			return nil
		},
	}
	benchmarkCmd.Flags().String("base-url", "", "OpenAI-compatible base URL")
	benchmarkCmd.Flags().String("model", "", "Model name to use for chat completion")
	benchmarkCmd.Flags().Int("requests", 3, "Number of sequential requests")
	benchmarkCmd.Flags().Duration("timeout", 60*time.Second, "HTTP timeout per request")

	expressCmd.AddCommand(detectCmd, recommendCmd, configCmd, preflightCmd, runCmd, healthcheckCmd, benchmarkCmd, newExpressRAGCommand())
	rootCmd.AddCommand(expressCmd)
}

func loadRecipeByID(root, id string) (recipe.Record, error) {
	recipesRoot, err := resolveRecipesRoot(root)
	if err != nil {
		return recipe.Record{}, err
	}
	registry, err := recipe.LoadRegistryFromDir(recipesRoot)
	if err != nil {
		return recipe.Record{}, err
	}
	record, ok := registry.Get(strings.TrimSpace(id))
	if !ok {
		return recipe.Record{}, fmt.Errorf("recipe %q not found", id)
	}
	return record, nil
}

func baseURLForRecipe(r recipe.Recipe) string {
	port := r.Runtime.Port
	if port <= 0 {
		port = 8000
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

func printPreflightReport(cmd *cobra.Command, report express.PreflightReport) {
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(writer, "CHECK\tSTATUS\tMESSAGE\tSUGGESTION\n")
	for _, check := range report.Checks {
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", check.Name, check.Status, check.Message, check.Suggestion)
	}
	_ = writer.Flush()
}

func printRunPlan(cmd *cobra.Command, plan express.RunPlan) {
	cmd.Printf("Recipe: %s\n", plan.RecipeID)
	cmd.Printf("Mode: %s\n", plan.Mode)
	cmd.Printf("Workdir: %s\n", plan.WorkDir)
	if len(plan.Command) > 0 {
		cmd.Printf("Command: %s\n", strings.Join(plan.Command, " "))
	}
	for _, note := range plan.Notes {
		cmd.Printf("Note: %s\n", note)
	}
}

func printRecommendations(cmd *cobra.Command, recs []express.Recommendation) {
	if len(recs) == 0 {
		cmd.Println("No matching Express inference recipes found.")
		return
	}
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "RECIPE\tCONFIDENCE\tSCORE\tREASONS")
	for _, rec := range recs {
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%d\t%s\n",
			rec.RecipeID,
			rec.Confidence,
			rec.Score,
			strings.Join(rec.Reasons, "; "),
		)
		if len(rec.Risks) > 0 {
			_, _ = fmt.Fprintf(writer, "\t\t\tRisks: %s\n", strings.Join(rec.Risks, "; "))
		}
		if len(rec.Fallbacks) > 0 {
			_, _ = fmt.Fprintf(writer, "\t\t\tFallbacks: %s\n", strings.Join(rec.Fallbacks, "; "))
		}
	}
	_ = writer.Flush()
}

func printConfigHints(cmd *cobra.Command, r recipe.Recipe) {
	cmd.Printf("Recipe: %s\n", r.ID)
	if r.Runtime.Port > 0 {
		cmd.Printf("BASE_URL=http://127.0.0.1:%d\n", r.Runtime.Port)
	}
	if r.Target.Runtime.Docker {
		cmd.Println("Runtime: docker-compose")
		cmd.Println("Command: ./build/las express run " + r.ID)
	}
	if r.Target.Runtime.Native {
		cmd.Println("Runtime fallback: native")
	}
	if r.Model.ID != "" {
		cmd.Printf("MODEL_ID=%s\n", r.Model.ID)
	}
	if len(r.Engine) > 0 {
		raw, err := json.MarshalIndent(r.Engine, "", "  ")
		if err == nil {
			cmd.Printf("Engine config:\n%s\n", string(raw))
		}
	}
}
