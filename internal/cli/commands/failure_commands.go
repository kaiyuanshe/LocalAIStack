package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/failure"
)

func RegisterFailureCommands(rootCmd *cobra.Command) {
	failureCmd := &cobra.Command{
		Use:   "failure",
		Short: "Inspect failure handling records",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List recent failure records",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")
			phase, _ := cmd.Flags().GetString("phase")
			category, _ := cmd.Flags().GetString("category")
			output, _ := cmd.Flags().GetString("output")

			events, err := failure.ListEvents("", limit, phase, category)
			if err != nil {
				return err
			}
			if strings.ToLower(strings.TrimSpace(output)) == "json" {
				payload, err := json.MarshalIndent(events, "", "  ")
				if err != nil {
					return err
				}
				cmd.Printf("%s\n", payload)
				return nil
			}
			if len(events) == 0 {
				cmd.Println("No failure records found.")
				return nil
			}

			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(writer, "ID\tTIME\tPHASE\tCATEGORY\tRETRYABLE\tMODULE\tMODEL")
			for _, event := range events {
				fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%t\t%s\t%s\n",
					event.ID,
					event.Timestamp,
					event.Phase,
					event.Classification.Category,
					event.Classification.Retryable,
					event.Module,
					event.Model,
				)
			}
			return writer.Flush()
		},
	}
	listCmd.Flags().Int("limit", 20, "Maximum number of failure events")
	listCmd.Flags().String("phase", "", "Filter by phase (install_planner/config_planner/smart_run/module_install/model_run)")
	listCmd.Flags().String("category", "", "Filter by category (auth/rate_limit/timeout/network/command_exit/...)")
	listCmd.Flags().String("output", "text", "Output format: text|json")

	showCmd := &cobra.Command{
		Use:   "show [event-id]",
		Short: "Show one failure event with suggested actions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			event, err := failure.FindEventByID("", args[0])
			if err != nil {
				return err
			}
			advice := failure.BuildAdvice(event.Classification)
			payload := map[string]any{
				"event":  event,
				"advice": advice,
			}
			encoded, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return err
			}
			cmd.Printf("%s\n", encoded)
			return nil
		},
	}

	failureCmd.AddCommand(listCmd, showCmd)
	rootCmd.AddCommand(failureCmd)
}
