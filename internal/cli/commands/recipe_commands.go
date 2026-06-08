package commands

import (
	"fmt"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
	"github.com/zhuangbiaowei/LocalAIStack/internal/recipe"
)

func RegisterRecipeCommands(rootCmd *cobra.Command) {
	recipeCmd := &cobra.Command{
		Use:     "recipes",
		Aliases: []string{"recipe"},
		Short:   "Manage LocalAIStack recipes",
	}

	var root string
	recipeCmd.PersistentFlags().StringVar(&root, "root", "", "recipes root directory")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List registered recipes",
		RunE: func(cmd *cobra.Command, args []string) error {
			recipesRoot, err := resolveRecipesRoot(root)
			if err != nil {
				return err
			}
			registry, err := recipe.LoadRegistryFromDir(recipesRoot)
			if err != nil {
				return err
			}
			records := registry.All()
			if len(records) == 0 {
				cmd.Println(i18n.T("No recipes found."))
				return nil
			}
			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(writer, "ID\tKIND\tTIER\tSTATUS\tPATH")
			for _, record := range records {
				relPath, err := filepath.Rel(recipesRoot, record.SourcePath)
				if err != nil {
					relPath = record.SourcePath
				}
				_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
					record.Recipe.ID,
					record.Recipe.Kind,
					record.Recipe.Tier,
					record.Recipe.Status,
					relPath,
				)
			}
			return writer.Flush()
		},
	}

	validateCmd := &cobra.Command{
		Use:   "validate [recipe-file]",
		Short: "Validate one recipe file or the full recipes registry",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				record, err := recipe.LoadRecord(args[0])
				if err != nil {
					return err
				}
				cmd.Printf("Recipe %s is valid (%s)\n", record.Recipe.ID, args[0])
				return nil
			}

			recipesRoot, err := resolveRecipesRoot(root)
			if err != nil {
				return err
			}
			registry, err := recipe.LoadRegistryFromDir(recipesRoot)
			if err != nil {
				return err
			}
			cmd.Printf("Validated %d recipes in %s\n", len(registry.All()), recipesRoot)
			return nil
		},
	}

	recipeCmd.AddCommand(listCmd, validateCmd)
	rootCmd.AddCommand(recipeCmd)
}

func resolveRecipesRoot(root string) (string, error) {
	if root != "" {
		return filepath.Abs(root)
	}
	return recipe.FindRecipesRoot()
}
