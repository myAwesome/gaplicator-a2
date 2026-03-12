package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"gapp/internal/generator"
)

var outputPath string

var buildCmd = &cobra.Command{
	Use:   "build <config.yaml>",
	Short: "Generate a web app from a YAML config",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := args[0]

		cfg, err := generator.ParseConfig(configPath)
		if err != nil {
			return fmt.Errorf("parse config: %w", err)
		}

		if errs := generator.ValidateConfig(cfg); len(errs) > 0 {
			fmt.Fprintf(os.Stderr, "config validation failed (%d error(s)):\n", len(errs))
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  - %v\n", e)
			}
			return fmt.Errorf("invalid config")
		}

		out := outputPath
		steps := []struct {
			name string
			fn   func() error
		}{
			{"Generating schema.sql", func() error {
				return writeFile(filepath.Join(out, "schema.sql"), generator.GenerateSchema(cfg.Models))
			}},
			{"Generating migrations", func() error {
				dir := filepath.Join(out, "migrations")
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				if err := writeFile(filepath.Join(dir, "001_initial.up.sql"), generator.GenerateMigrationUp(cfg.Models)); err != nil {
					return err
				}
				return writeFile(filepath.Join(dir, "001_initial.down.sql"), generator.GenerateMigrationDown(cfg.Models))
			}},
			{"Generating models", func() error {
				dir := filepath.Join(out, "models")
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				return writeFile(filepath.Join(dir, "models.go"), generator.GenerateGORMModels(cfg.Models, "models"))
			}},
			{"Generating routes", func() error {
				dir := filepath.Join(out, "routes")
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				return writeFile(filepath.Join(dir, "routes.go"), generator.GenerateGinRoutes(cfg.Models, "routes", cfg.App.Name+"/models"))
			}},
			{"Generating main.go", func() error {
				return writeFile(filepath.Join(out, "main.go"), generator.GenerateMain(cfg, cfg.App.Name))
			}},
			{"Generating docker-compose.yml", func() error {
				return writeFile(filepath.Join(out, "docker-compose.yml"), generator.GenerateDockerCompose(cfg))
			}},
			{"Generating go.mod", func() error {
				return writeFile(filepath.Join(out, "go.mod"), generator.GenerateGoMod(cfg))
			}},
			{"Generating .env", func() error {
				return writeFile(filepath.Join(out, ".env"), generator.GenerateEnv(cfg))
			}},
			{"Generating dev.sh", func() error {
				return writeExecutable(filepath.Join(out, "dev.sh"), generator.GenerateDevScript(cfg))
			}},
			{"Generating shutdown.sh", func() error {
				return writeExecutable(filepath.Join(out, "shutdown.sh"), generator.GenerateShutdownScript())
			}},
		}

		if err := os.MkdirAll(out, 0755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}

		fmt.Printf("Building %s → %s\n\n", cfg.App.Name, out)
		for i, step := range steps {
			fmt.Printf("  [%d/%d] %s...\n", i+1, len(steps), step.name)
			if err := step.fn(); err != nil {
				return fmt.Errorf("%s: %w", step.name, err)
			}
		}
		fmt.Printf("\nBuild complete! Artifacts written to: %s\n", out)
		return nil
	},
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func writeExecutable(path, content string) error {
	return os.WriteFile(path, []byte(content), 0755)
}

func init() {
	buildCmd.Flags().StringVarP(&outputPath, "output", "o", "dist", "Output directory for generated files")
}
