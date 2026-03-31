package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/myAwesome/gaplicator/internal/generator"
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

		var backendSteps []struct {
			name string
			fn   func() error
		}

		if cfg.App.Server == "node" {
			backendSteps = []struct {
				name string
				fn   func() error
			}{
				{"Generating package.json", func() error {
					content, err := generator.GenerateNodePackageJSON(cfg)
					if err != nil {
						return err
					}
					return writeFile(filepath.Join(out, "package.json"), content)
				}},
				{"Generating index.js", func() error {
					content, err := generator.GenerateNodeIndex(cfg)
					if err != nil {
						return err
					}
					return writeFile(filepath.Join(out, "index.js"), content)
				}},
				{"Generating routes.js", func() error {
					content, err := generator.GenerateNodeRoutes(cfg)
					if err != nil {
						return err
					}
					return writeFile(filepath.Join(out, "routes.js"), content)
				}},
				{"Generating auth.js", func() error {
					if cfg.Auth == nil {
						return nil
					}
					content, err := generator.GenerateNodeAuth(cfg)
					if err != nil {
						return err
					}
					return writeFile(filepath.Join(out, "auth.js"), content)
				}},
				{"Generating prisma/schema.prisma", func() error {
					dir := filepath.Join(out, "prisma")
					if err := os.MkdirAll(dir, 0755); err != nil {
						return err
					}
					content, err := generator.GenerateNodePrismaSchema(cfg)
					if err != nil {
						return err
					}
					return writeFile(filepath.Join(dir, "schema.prisma"), content)
				}},
			}
		} else {
			backendSteps = []struct {
				name string
				fn   func() error
			}{
				{"Generating migrations", func() error {
					dir := filepath.Join(out, "migrations")
					if err := os.MkdirAll(dir, 0755); err != nil {
						return err
					}
					return writeFile(filepath.Join(dir, "001_initial.up.sql"), generator.GenerateMigrationUp(cfg.Models, cfg.Database.Driver))
				}},
				{"Generating models", func() error {
					dir := filepath.Join(out, "models")
					if err := os.MkdirAll(dir, 0755); err != nil {
						return err
					}
					return writeFile(filepath.Join(dir, "models.go"), generator.GenerateGORMModels(cfg.Models, "models", cfg.Auth))
				}},
				{"Generating routes", func() error {
					dir := filepath.Join(out, "routes")
					if err := os.MkdirAll(dir, 0755); err != nil {
						return err
					}
					return writeFile(filepath.Join(dir, "routes.go"), generator.GenerateGinRoutes(cfg.Models, "routes", cfg.App.Name+"/models", cfg.Database.Driver == "mysql"))
				}},
				{"Generating main.go", func() error {
					content, err := generator.GenerateMain(cfg, cfg.App.Name)
					if err != nil {
						return err
					}
					return writeFile(filepath.Join(out, "main.go"), content)
				}},
				{"Generating auth.go", func() error {
					if cfg.Auth == nil {
						return nil
					}
					content, err := generator.GenerateAuthGo(cfg, cfg.App.Name)
					if err != nil {
						return err
					}
					return writeFile(filepath.Join(out, "auth.go"), content)
				}},
				{"Generating go.mod", func() error {
					content, err := generator.GenerateGoMod(cfg)
					if err != nil {
						return err
					}
					return writeFile(filepath.Join(out, "go.mod"), content)
				}},
			}
		}

		devShStep := struct {
			name string
			fn   func() error
		}{"Generating dev.sh", func() error {
			var content string
			var err error
			if cfg.App.Server == "node" {
				content, err = generator.GenerateNodeDevScript(cfg)
			} else {
				content, err = generator.GenerateDevScript(cfg)
			}
			if err != nil {
				return err
			}
			return writeExecutable(filepath.Join(out, "dev.sh"), content)
		}}

		steps := []struct {
			name string
			fn   func() error
		}{}
		steps = append(steps, backendSteps...)
		steps = append(steps, struct {
			name string
			fn   func() error
		}{"Generating docker-compose.yml", func() error {
			content, err := generator.GenerateDockerCompose(cfg)
			if err != nil {
				return err
			}
			return writeFile(filepath.Join(out, "docker-compose.yml"), content)
		}})
		steps = append(steps, struct {
			name string
			fn   func() error
		}{"Generating .env", func() error {
			content, err := generator.GenerateEnv(cfg)
			if err != nil {
				return err
			}
			return writeFile(filepath.Join(out, ".env"), content)
		}})
		steps = append(steps, devShStep)
		steps = append(steps, struct {
			name string
			fn   func() error
		}{"Generating shutdown.sh", func() error {
			content, err := generator.GenerateShutdownScript()
			if err != nil {
				return err
			}
			return writeExecutable(filepath.Join(out, "shutdown.sh"), content)
		}})
		steps = append(steps, struct {
			name string
			fn   func() error
		}{"Generating README.md", func() error {
			content, err := generator.GenerateReadme(cfg)
			if err != nil {
				return err
			}
			return writeFile(filepath.Join(out, "README.md"), content)
		}})
		steps = append(steps, struct {
			name string
			fn   func() error
		}{"Generating client", func() error {
			clientDir := filepath.Join(out, "client")
			srcDir := filepath.Join(clientDir, "src")
			dirs := []string{
				filepath.Join(srcDir, "types"),
				filepath.Join(srcDir, "api"),
				filepath.Join(srcDir, "pages"),
			}
			if cfg.Auth != nil {
				dirs = append(dirs, filepath.Join(srcDir, "context"))
			}
			for _, dir := range dirs {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
			}
			static := map[string]string{
				filepath.Join(clientDir, "package.json"):   generator.GenerateReactPackageJSON(cfg),
				filepath.Join(clientDir, "index.html"):     generator.GenerateReactIndexHTML(cfg),
				filepath.Join(clientDir, "vite.config.ts"): generator.GenerateReactViteConfig(cfg),
				filepath.Join(clientDir, "tsconfig.json"):  generator.GenerateReactTsConfig(),
				filepath.Join(srcDir, "main.tsx"):          generator.GenerateReactMain(),
				filepath.Join(srcDir, "app.css"):           generator.GenerateReactAppCSS(),
				filepath.Join(srcDir, "App.tsx"):           generator.GenerateReactApp(cfg.Models, cfg.Auth != nil),
			}
			for path, content := range static {
				if err := writeFile(path, content); err != nil {
					return err
				}
			}
			if cfg.Auth != nil {
				authFiles := map[string]string{
					filepath.Join(srcDir, "context", "AuthContext.tsx"): generator.GenerateReactAuthContext(),
					filepath.Join(srcDir, "api", "auth.ts"):            generator.GenerateReactAuthAPI(cfg),
					filepath.Join(srcDir, "pages", "LoginPage.tsx"):    generator.GenerateReactLoginPage(cfg),
					filepath.Join(srcDir, "pages", "RegisterPage.tsx"): generator.GenerateReactRegisterPage(cfg),
				}
				for path, content := range authFiles {
					if err := writeFile(path, content); err != nil {
						return err
					}
				}
			}
			for _, m := range cfg.Models {
				base := generator.ModelFileBasename(m)
				structName := generator.ModelStructName(m)
				if err := writeFile(filepath.Join(srcDir, "types", base+".ts"), generator.GenerateReactTypes(m, cfg.Models)); err != nil {
					return err
				}
				if err := writeFile(filepath.Join(srcDir, "api", base+".ts"), generator.GenerateReactAPI(m, cfg.Auth != nil)); err != nil {
					return err
				}
				if err := writeFile(filepath.Join(srcDir, "pages", structName+"Page.tsx"), generator.GenerateReactPage(m, cfg.Models)); err != nil {
					return err
				}
			}
			return nil
		}})

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
