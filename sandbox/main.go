package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	Database DatabaseConfig `yaml:"database"`
	Models   []Model        `yaml:"models"`
}

type AppConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Host string `yaml:"host"`
	Name string `yaml:"name"`
}

type Model struct {
	Name   string  `yaml:"name"`
	Fields []Field `yaml:"fields"`
}

type Field struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Required   bool   `yaml:"required"`
	Unique     bool   `yaml:"unique"`
	Default    any    `yaml:"default"`
	References string `yaml:"references"` // e.g. "subjects.id"
}

var validTypeRe = regexp.MustCompile(
	`^(int|bigint|smallint|text|boolean|bool|date|datetime|timestamp|uuid|float|double|` +
		`varchar\(\d+\)|char\(\d+\)|decimal\(\d+,\s*\d+\))$`,
)

func ValidateConfig(cfg *Config) []error {
	var errs []error

	// Required top-level fields
	if cfg.App.Name == "" {
		errs = append(errs, fmt.Errorf("app.name is required"))
	}
	if cfg.App.Port == 0 {
		errs = append(errs, fmt.Errorf("app.port is required"))
	}
	if cfg.Database.Host == "" {
		errs = append(errs, fmt.Errorf("database.host is required"))
	}
	if cfg.Database.Name == "" {
		errs = append(errs, fmt.Errorf("database.name is required"))
	}
	if len(cfg.Models) == 0 {
		errs = append(errs, fmt.Errorf("at least one model is required"))
	}

	// Build model name set for reference validation
	modelNames := make(map[string]bool, len(cfg.Models))
	for _, m := range cfg.Models {
		if m.Name != "" {
			modelNames[m.Name] = true
		}
	}

	for mi, m := range cfg.Models {
		prefix := fmt.Sprintf("models[%d]", mi)
		if m.Name == "" {
			errs = append(errs, fmt.Errorf("%s.name is required", prefix))
			prefix = fmt.Sprintf("models[%d]", mi) // keep index-based prefix
		} else {
			prefix = fmt.Sprintf("model %q", m.Name)
		}

		if len(m.Fields) == 0 {
			errs = append(errs, fmt.Errorf("%s: at least one field is required", prefix))
		}

		for fi, f := range m.Fields {
			fprefix := fmt.Sprintf("%s field[%d]", prefix, fi)
			if f.Name != "" {
				fprefix = fmt.Sprintf("%s field %q", prefix, f.Name)
			}

			if f.Name == "" {
				errs = append(errs, fmt.Errorf("%s: name is required", fprefix))
			}

			if f.Type == "" {
				errs = append(errs, fmt.Errorf("%s: type is required", fprefix))
			} else if !validTypeRe.MatchString(strings.ToLower(f.Type)) {
				errs = append(errs, fmt.Errorf("%s: unknown type %q", fprefix, f.Type))
			}

			if f.References != "" {
				parts := strings.SplitN(f.References, ".", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					errs = append(errs, fmt.Errorf("%s: references %q must be in \"model.field\" format", fprefix, f.References))
				} else if !modelNames[parts[0]] {
					errs = append(errs, fmt.Errorf("%s: references unknown model %q", fprefix, parts[0]))
				}
			}
		}
	}

	return errs
}

// GenerateSchema returns a schema.sql string for all models in dependency order.
func GenerateSchema(models []Model) string {
	sorted := topoSort(models)
	var sb strings.Builder
	for i, m := range sorted {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(tableSQL(m))
	}
	return sb.String()
}

func tableSQL(m Model) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "CREATE TABLE %s (\n", m.Name)
	sb.WriteString("    id SERIAL PRIMARY KEY")
	for _, f := range m.Fields {
		sb.WriteString(",\n")
		fmt.Fprintf(&sb, "    %s %s", f.Name, strings.ToUpper(f.Type))
		if f.Required {
			sb.WriteString(" NOT NULL")
		}
		if f.Unique {
			sb.WriteString(" UNIQUE")
		}
		if f.Default != nil {
			fmt.Fprintf(&sb, " DEFAULT %s", formatDefault(f.Default))
		}
		if f.References != "" {
			parts := strings.SplitN(f.References, ".", 2)
			fmt.Fprintf(&sb, " REFERENCES %s(%s)", parts[0], parts[1])
		}
	}
	sb.WriteString("\n);\n")
	return sb.String()
}

func formatDefault(v any) string {
	switch val := v.(type) {
	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"
	case string:
		return fmt.Sprintf("'%s'", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// topoSort orders models so that referenced tables are created before their dependants.
func topoSort(models []Model) []Model {
	index := make(map[string]int, len(models))
	for i, m := range models {
		index[m.Name] = i
	}
	visited := make(map[string]bool, len(models))
	result := make([]Model, 0, len(models))

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		i, ok := index[name]
		if !ok {
			return
		}
		for _, f := range models[i].Fields {
			if f.References != "" {
				parts := strings.SplitN(f.References, ".", 2)
				if parts[0] != name {
					visit(parts[0])
				}
			}
		}
		result = append(result, models[i])
	}

	for _, m := range models {
		visit(m.Name)
	}
	return result
}

func ParseConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	return &cfg, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: web-app-gen <config.yaml>")
	}

	cfg, err := ParseConfig(os.Args[1])
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if errs := ValidateConfig(cfg); len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "config validation failed (%d error(s)):\n", len(errs))
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}
		os.Exit(1)
	}

	fmt.Printf("App: %s (port %d)\n", cfg.App.Name, cfg.App.Port)
	fmt.Printf("DB:  %s @ %s\n", cfg.Database.Name, cfg.Database.Host)
	fmt.Printf("Models (%d):\n", len(cfg.Models))
	for _, m := range cfg.Models {
		fmt.Printf("  %s (%d fields)\n", m.Name, len(m.Fields))
		for _, f := range m.Fields {
			fmt.Printf("    - %s %s", f.Name, f.Type)
			if f.Required {
				fmt.Print(" [required]")
			}
			if f.Unique {
				fmt.Print(" [unique]")
			}
			if f.Default != nil {
				fmt.Printf(" [default: %v]", f.Default)
			}
			if f.References != "" {
				fmt.Printf(" -> %s", f.References)
			}
			fmt.Println()
		}
	}

	schema := GenerateSchema(cfg.Models)
	if err := os.WriteFile("schema.sql", []byte(schema), 0644); err != nil {
		log.Fatalf("write schema.sql: %v", err)
	}
	fmt.Println("\nGenerated schema.sql")
}
