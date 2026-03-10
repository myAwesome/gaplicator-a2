package main

import (
	"fmt"
	"log"
	"os"

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
			if f.References != "" {
				fmt.Printf(" -> %s", f.References)
			}
			fmt.Println()
		}
	}
}
