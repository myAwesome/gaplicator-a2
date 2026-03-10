package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var outputPath string

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the project",
	RunE: func(cmd *cobra.Command, args []string) error {
		steps := []string{
			"Resolving dependencies",
			"Compiling source files",
			"Running tests",
			"Optimizing bundle",
			"Writing output",
		}

		fmt.Printf("Starting build → %s\n\n", outputPath)
		for i, step := range steps {
			fmt.Printf("  [%d/%d] %s...\n", i+1, len(steps), step)
			time.Sleep(300 * time.Millisecond)
		}
		fmt.Printf("\nBuild complete! Artifacts written to: %s\n", outputPath)
		return nil
	},
}

func init() {
	buildCmd.Flags().StringVarP(&outputPath, "output", "o", "dist", "Output path for the build")
}
