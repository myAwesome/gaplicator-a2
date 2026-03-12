package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

//go:embed templates/main.go.tmpl
var mainTmpl string

//go:embed templates/docker-compose.yml.tmpl
var dockerComposeTmpl string

//go:embed templates/go.mod.tmpl
var goModTmpl string

//go:embed templates/.env.tmpl
var envTmpl string

//go:embed templates/dev.sh.tmpl
var devShTmpl string

//go:embed templates/shutdown.sh.tmpl
var shutdownShTmpl string

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
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
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

// GenerateMigrationUp returns the UP migration SQL (CREATE TABLE statements in dependency order).
func GenerateMigrationUp(models []Model) string {
	return GenerateSchema(models)
}

// GenerateMigrationDown returns the DOWN migration SQL (DROP TABLE statements in reverse dependency order).
func GenerateMigrationDown(models []Model) string {
	sorted := topoSort(models)
	var sb strings.Builder
	for i := len(sorted) - 1; i >= 0; i-- {
		fmt.Fprintf(&sb, "DROP TABLE IF EXISTS %s;\n", sorted[i].Name)
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

// toSingular converts a plural snake_case table name to singular.
func toSingular(s string) string {
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "sses") || strings.HasSuffix(s, "xes") || strings.HasSuffix(s, "ches") || strings.HasSuffix(s, "shes") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}
	return s
}

// goAcronyms lists lowercase words that should be fully uppercased in Go identifiers.
var goAcronyms = map[string]string{
	"id": "ID", "url": "URL", "uri": "URI", "api": "API",
	"http": "HTTP", "https": "HTTPS", "sql": "SQL", "db": "DB",
	"uuid": "UUID", "ip": "IP",
}

// toPascalCase converts snake_case to PascalCase, honoring Go acronym conventions.
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	var sb strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		if upper, ok := goAcronyms[strings.ToLower(p)]; ok {
			sb.WriteString(upper)
		} else {
			sb.WriteString(strings.ToUpper(p[:1]) + p[1:])
		}
	}
	return sb.String()
}

// sqlTypeToGo maps a SQL type to the corresponding Go type.
func sqlTypeToGo(sqlType string) string {
	lower := strings.ToLower(sqlType)
	switch {
	case strings.HasPrefix(lower, "varchar"), strings.HasPrefix(lower, "char"), lower == "text", lower == "uuid":
		return "string"
	case lower == "int", lower == "smallint":
		return "int"
	case lower == "bigint":
		return "int64"
	case lower == "boolean", lower == "bool":
		return "bool"
	case lower == "date", lower == "datetime", lower == "timestamp":
		return "time.Time"
	case lower == "float", lower == "double":
		return "float64"
	case strings.HasPrefix(lower, "decimal"):
		return "float64"
	default:
		return "interface{}"
	}
}

// extractSize returns the N from varchar(N) / char(N), or "" if not applicable.
func extractSize(sqlType string) string {
	lower := strings.ToLower(sqlType)
	if strings.HasPrefix(lower, "varchar(") || strings.HasPrefix(lower, "char(") {
		start := strings.Index(lower, "(")
		end := strings.Index(lower, ")")
		if start >= 0 && end > start {
			return sqlType[start+1 : end]
		}
	}
	return ""
}

// buildGORMTag builds the `gorm:"..."` tag value for a field.
func buildFieldTags(f Field) string {
	var parts []string
	parts = append(parts, "column:"+f.Name)
	if size := extractSize(f.Type); size != "" {
		parts = append(parts, "size:"+size)
	}
	if f.Required {
		parts = append(parts, "not null")
	}
	if f.Unique {
		parts = append(parts, "uniqueIndex")
	}
	if f.Default != nil {
		parts = append(parts, "default:"+formatDefault(f.Default))
	}
	return `gorm:"` + strings.Join(parts, ";") + `" json:"` + f.Name + `"`
}

// GenerateGORMModels returns Go source code with GORM model structs for all models.
func GenerateGORMModels(models []Model, pkgName string) string {
	// Build struct name lookup: table name → struct name.
	structNames := make(map[string]string, len(models))
	for _, m := range models {
		structNames[m.Name] = toPascalCase(toSingular(m.Name))
	}

	var sb strings.Builder
	sb.WriteString("package " + pkgName + "\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"time\"\n\n")
	sb.WriteString("\t\"gorm.io/gorm\"\n")
	sb.WriteString(")\n")

	sb.WriteString("\ntype Base struct {\n")
	sb.WriteString("\tID        uint           `gorm:\"primarykey\" json:\"id\"`\n")
	sb.WriteString("\tCreatedAt time.Time      `json:\"created_at\"`\n")
	sb.WriteString("\tUpdatedAt time.Time      `json:\"updated_at\"`\n")
	sb.WriteString("\tDeletedAt gorm.DeletedAt `gorm:\"index\" json:\"deleted_at,omitempty\"`\n")
	sb.WriteString("}\n")

	for _, m := range models {
		structName := structNames[m.Name]
		sb.WriteString("\ntype " + structName + " struct {\n")
		sb.WriteString("\tBase\n")

		for _, f := range m.Fields {
			fieldName := toPascalCase(f.Name)
			goType := sqlTypeToGo(f.Type)
			tags := buildFieldTags(f)
			fmt.Fprintf(&sb, "\t%-20s %-12s `%s`\n", fieldName, goType, tags)

			if f.References != "" {
				parts := strings.SplitN(f.References, ".", 2)
				if assocStruct, ok := structNames[parts[0]]; ok {
					assocField := strings.TrimSuffix(fieldName, "ID")
					if assocField == fieldName {
						assocField = strings.TrimSuffix(fieldName, "Id")
					}
					if assocField == fieldName {
						assocField = assocStruct
					}
					assocJsonName := strings.TrimSuffix(f.Name, "_id")
					if assocJsonName == f.Name {
						assocJsonName = strings.TrimSuffix(f.Name, "_ID")
					}
					fmt.Fprintf(&sb, "\t%-20s %-12s `gorm:\"foreignKey:%s\" json:\"%s\"`\n", assocField, assocStruct, fieldName, assocJsonName)
				}
			}
		}

		sb.WriteString("}\n")
	}

	return sb.String()
}

// ModelStructName returns the PascalCase singular struct name for a model.
func ModelStructName(m Model) string {
	return toPascalCase(toSingular(m.Name))
}

// ModelFileBasename returns the singular snake_case file basename for a model.
func ModelFileBasename(m Model) string {
	return toSingular(m.Name)
}

// sqlTypeToTS maps a SQL type to the corresponding TypeScript type.
func sqlTypeToTS(sqlType string) string {
	lower := strings.ToLower(sqlType)
	switch {
	case strings.HasPrefix(lower, "varchar"), strings.HasPrefix(lower, "char"), lower == "text", lower == "uuid":
		return "string"
	case lower == "int", lower == "bigint", lower == "smallint":
		return "number"
	case lower == "boolean", lower == "bool":
		return "boolean"
	case lower == "date", lower == "datetime", lower == "timestamp":
		return "string"
	case lower == "float", lower == "double":
		return "number"
	case strings.HasPrefix(lower, "decimal"):
		return "number"
	default:
		return "unknown"
	}
}

func tsInputDefault(sqlType string) string {
	lower := strings.ToLower(sqlType)
	switch {
	case lower == "boolean", lower == "bool":
		return "false"
	case lower == "int", lower == "bigint", lower == "smallint", lower == "float", lower == "double":
		return "0"
	case strings.HasPrefix(lower, "decimal"):
		return "0"
	default:
		return "''"
	}
}

func tsInputType(sqlType string) string {
	lower := strings.ToLower(sqlType)
	switch {
	case lower == "boolean", lower == "bool":
		return "checkbox"
	case lower == "int", lower == "bigint", lower == "smallint", lower == "float", lower == "double":
		return "number"
	case strings.HasPrefix(lower, "decimal"):
		return "number"
	case lower == "date":
		return "date"
	default:
		return "text"
	}
}

func GenerateReactPackageJSON(cfg *Config) string {
	return fmt.Sprintf(`{
  "name": "%s-client",
  "version": "0.0.1",
  "private": true,
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "react-router-dom": "^6.28.0"
  },
  "devDependencies": {
    "@types/react": "^18.3.12",
    "@types/react-dom": "^18.3.1",
    "@vitejs/plugin-react": "^4.3.4",
    "typescript": "~5.6.2",
    "vite": "^6.0.5"
  }
}
`, cfg.App.Name)
}

func GenerateReactIndexHTML(cfg *Config) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>%s</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
`, cfg.App.Name)
}

func GenerateReactViteConfig(cfg *Config) string {
	var sb strings.Builder
	sb.WriteString("import { defineConfig } from 'vite'\n")
	sb.WriteString("import react from '@vitejs/plugin-react'\n\n")
	sb.WriteString("export default defineConfig({\n")
	sb.WriteString("  plugins: [react()],\n")
	sb.WriteString("  server: {\n")
	sb.WriteString("    proxy: {\n")
	for _, m := range cfg.Models {
		fmt.Fprintf(&sb, "      '/%s': 'http://localhost:%d',\n", m.Name, cfg.App.Port)
	}
	sb.WriteString("    },\n")
	sb.WriteString("  },\n")
	sb.WriteString("})\n")
	return sb.String()
}

func GenerateReactTsConfig() string {
	return `{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true
  },
  "include": ["src"]
}
`
}

func GenerateReactMain() string {
	return `import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
`
}

func GenerateReactApp(models []Model) string {
	var sb strings.Builder
	sb.WriteString("import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom';\n")
	for _, m := range models {
		structName := toPascalCase(toSingular(m.Name))
		fmt.Fprintf(&sb, "import %sPage from './pages/%sPage';\n", structName, structName)
	}
	sb.WriteString("\nexport default function App() {\n")
	sb.WriteString("  return (\n")
	sb.WriteString("    <BrowserRouter>\n")
	sb.WriteString("      <nav>\n")
	for i, m := range models {
		structName := toPascalCase(toSingular(m.Name))
		if i > 0 {
			sb.WriteString("        {' | '}\n")
		}
		fmt.Fprintf(&sb, "        <NavLink to=\"/%s\">%s</NavLink>\n", m.Name, structName)
	}
	sb.WriteString("      </nav>\n")
	sb.WriteString("      <main>\n")
	sb.WriteString("        <Routes>\n")
	for _, m := range models {
		structName := toPascalCase(toSingular(m.Name))
		fmt.Fprintf(&sb, "          <Route path=\"/%s\" element={<%sPage />} />\n", m.Name, structName)
	}
	sb.WriteString("        </Routes>\n")
	sb.WriteString("      </main>\n")
	sb.WriteString("    </BrowserRouter>\n")
	sb.WriteString("  );\n")
	sb.WriteString("}\n")
	return sb.String()
}

func GenerateReactTypes(m Model) string {
	structName := toPascalCase(toSingular(m.Name))
	var sb strings.Builder

	fmt.Fprintf(&sb, "export interface %s {\n", structName)
	sb.WriteString("  id: number;\n")
	for _, f := range m.Fields {
		tsType := sqlTypeToTS(f.Type)
		if f.Required {
			fmt.Fprintf(&sb, "  %s: %s;\n", f.Name, tsType)
		} else {
			fmt.Fprintf(&sb, "  %s?: %s;\n", f.Name, tsType)
		}
	}
	sb.WriteString("  created_at: string;\n")
	sb.WriteString("  updated_at: string;\n")
	sb.WriteString("  deleted_at?: string;\n")
	sb.WriteString("}\n\n")

	fmt.Fprintf(&sb, "export type Create%sInput = {\n", structName)
	for _, f := range m.Fields {
		tsType := sqlTypeToTS(f.Type)
		fmt.Fprintf(&sb, "  %s: %s;\n", f.Name, tsType)
	}
	sb.WriteString("};\n")

	return sb.String()
}

func GenerateReactAPI(m Model) string {
	singular := toSingular(m.Name)
	structName := toPascalCase(singular)
	pluralName := toPascalCase(m.Name)

	var sb strings.Builder
	fmt.Fprintf(&sb, "import type { %s, Create%sInput } from '../types/%s';\n\n", structName, structName, singular)
	fmt.Fprintf(&sb, "const BASE = '/%s';\n\n", m.Name)

	fmt.Fprintf(&sb, "export async function list%s(): Promise<%s[]> {\n", pluralName, structName)
	sb.WriteString("  const res = await fetch(BASE);\n")
	sb.WriteString("  if (!res.ok) throw new Error(await res.text());\n")
	sb.WriteString("  return res.json();\n")
	sb.WriteString("}\n\n")

	fmt.Fprintf(&sb, "export async function get%s(id: number): Promise<%s> {\n", structName, structName)
	sb.WriteString("  const res = await fetch(`${BASE}/${id}`);\n")
	sb.WriteString("  if (!res.ok) throw new Error(await res.text());\n")
	sb.WriteString("  return res.json();\n")
	sb.WriteString("}\n\n")

	fmt.Fprintf(&sb, "export async function create%s(data: Create%sInput): Promise<%s> {\n", structName, structName, structName)
	sb.WriteString("  const res = await fetch(BASE, {\n")
	sb.WriteString("    method: 'POST',\n")
	sb.WriteString("    headers: { 'Content-Type': 'application/json' },\n")
	sb.WriteString("    body: JSON.stringify(data),\n")
	sb.WriteString("  });\n")
	sb.WriteString("  if (!res.ok) throw new Error(await res.text());\n")
	sb.WriteString("  return res.json();\n")
	sb.WriteString("}\n\n")

	fmt.Fprintf(&sb, "export async function update%s(id: number, data: Partial<Create%sInput>): Promise<%s> {\n", structName, structName, structName)
	sb.WriteString("  const res = await fetch(`${BASE}/${id}`, {\n")
	sb.WriteString("    method: 'PUT',\n")
	sb.WriteString("    headers: { 'Content-Type': 'application/json' },\n")
	sb.WriteString("    body: JSON.stringify(data),\n")
	sb.WriteString("  });\n")
	sb.WriteString("  if (!res.ok) throw new Error(await res.text());\n")
	sb.WriteString("  return res.json();\n")
	sb.WriteString("}\n\n")

	fmt.Fprintf(&sb, "export async function delete%s(id: number): Promise<void> {\n", structName)
	sb.WriteString("  const res = await fetch(`${BASE}/${id}`, { method: 'DELETE' });\n")
	sb.WriteString("  if (!res.ok) throw new Error(await res.text());\n")
	sb.WriteString("}\n")

	return sb.String()
}

func GenerateReactPage(m Model) string {
	singular := toSingular(m.Name)
	structName := toPascalCase(singular)
	pluralName := toPascalCase(m.Name)
	componentName := structName + "Page"

	var sb strings.Builder

	fmt.Fprintf(&sb, "import { useState, useEffect } from 'react';\n")
	fmt.Fprintf(&sb, "import type { %s, Create%sInput } from '../types/%s';\n", structName, structName, singular)
	fmt.Fprintf(&sb, "import { list%s, create%s, update%s, delete%s } from '../api/%s';\n\n", pluralName, structName, structName, structName, singular)

	fmt.Fprintf(&sb, "const EMPTY_FORM: Create%sInput = {\n", structName)
	for _, f := range m.Fields {
		fmt.Fprintf(&sb, "  %s: %s,\n", f.Name, tsInputDefault(f.Type))
	}
	sb.WriteString("};\n\n")

	fmt.Fprintf(&sb, "export default function %s() {\n", componentName)
	fmt.Fprintf(&sb, "  const [items, setItems] = useState<%s[]>([]);\n", structName)
	fmt.Fprintf(&sb, "  const [editing, setEditing] = useState<%s | null>(null);\n", structName)
	fmt.Fprintf(&sb, "  const [form, setForm] = useState<Create%sInput>(EMPTY_FORM);\n", structName)
	sb.WriteString("  const [showForm, setShowForm] = useState(false);\n\n")
	sb.WriteString("  useEffect(() => { load(); }, []);\n\n")

	fmt.Fprintf(&sb, "  async function load() {\n")
	fmt.Fprintf(&sb, "    try { setItems(await list%s()); } catch (e) { console.error(e); }\n", pluralName)
	sb.WriteString("  }\n\n")

	sb.WriteString("  function openCreate() {\n")
	sb.WriteString("    setEditing(null); setForm(EMPTY_FORM); setShowForm(true);\n")
	sb.WriteString("  }\n\n")

	fmt.Fprintf(&sb, "  function openEdit(item: %s) {\n", structName)
	sb.WriteString("    setEditing(item);\n")
	sb.WriteString("    setForm({\n")
	for _, f := range m.Fields {
		if f.Required {
			fmt.Fprintf(&sb, "      %s: item.%s,\n", f.Name, f.Name)
		} else {
			fmt.Fprintf(&sb, "      %s: item.%s ?? %s,\n", f.Name, f.Name, tsInputDefault(f.Type))
		}
	}
	sb.WriteString("    });\n")
	sb.WriteString("    setShowForm(true);\n")
	sb.WriteString("  }\n\n")

	sb.WriteString("  async function handleSubmit(e: React.FormEvent) {\n")
	sb.WriteString("    e.preventDefault();\n")
	sb.WriteString("    try {\n")
	fmt.Fprintf(&sb, "      if (editing) await update%s(editing.id, form);\n", structName)
	fmt.Fprintf(&sb, "      else await create%s(form);\n", structName)
	sb.WriteString("      setShowForm(false); load();\n")
	sb.WriteString("    } catch (e) { console.error(e); }\n")
	sb.WriteString("  }\n\n")

	sb.WriteString("  async function handleDelete(id: number) {\n")
	sb.WriteString("    if (!confirm('Delete?')) return;\n")
	fmt.Fprintf(&sb, "    try { await delete%s(id); load(); } catch (e) { console.error(e); }\n", structName)
	sb.WriteString("  }\n\n")

	sb.WriteString("  return (\n")
	sb.WriteString("    <div>\n")
	fmt.Fprintf(&sb, "      <h1>%s</h1>\n", m.Name)
	sb.WriteString("      <button onClick={openCreate}>+ New</button>\n")
	sb.WriteString("      {showForm && (\n")
	sb.WriteString("        <form onSubmit={handleSubmit}>\n")
	for _, f := range m.Fields {
		inputType := tsInputType(f.Type)
		if inputType == "checkbox" {
			fmt.Fprintf(&sb, "          <label>%s <input type=\"checkbox\" checked={form.%s as boolean} onChange={e => setForm({...form, %s: e.target.checked})} /></label>\n", f.Name, f.Name, f.Name)
		} else if inputType == "number" {
			reqAttr := ""
			if f.Required {
				reqAttr = " required"
			}
			fmt.Fprintf(&sb, "          <label>%s <input type=\"number\" value={form.%s as number} onChange={e => setForm({...form, %s: Number(e.target.value)})}%s /></label>\n", f.Name, f.Name, f.Name, reqAttr)
		} else {
			reqAttr := ""
			if f.Required {
				reqAttr = " required"
			}
			fmt.Fprintf(&sb, "          <label>%s <input type=\"%s\" value={form.%s as string} onChange={e => setForm({...form, %s: e.target.value})}%s /></label>\n", f.Name, inputType, f.Name, f.Name, reqAttr)
		}
	}
	sb.WriteString("          <button type=\"submit\">{editing ? 'Save' : 'Create'}</button>\n")
	sb.WriteString("          <button type=\"button\" onClick={() => setShowForm(false)}>Cancel</button>\n")
	sb.WriteString("        </form>\n")
	sb.WriteString("      )}\n")
	sb.WriteString("      <table>\n")
	sb.WriteString("        <thead><tr><th>id</th>")
	for _, f := range m.Fields {
		fmt.Fprintf(&sb, "<th>%s</th>", f.Name)
	}
	sb.WriteString("<th></th></tr></thead>\n")
	sb.WriteString("        <tbody>\n")
	sb.WriteString("          {items.map(item => (\n")
	sb.WriteString("            <tr key={item.id}>\n")
	sb.WriteString("              <td>{item.id}</td>\n")
	for _, f := range m.Fields {
		if sqlTypeToTS(f.Type) == "boolean" {
			fmt.Fprintf(&sb, "              <td>{item.%s ? 'yes' : 'no'}</td>\n", f.Name)
		} else {
			fmt.Fprintf(&sb, "              <td>{item.%s}</td>\n", f.Name)
		}
	}
	sb.WriteString("              <td>\n")
	sb.WriteString("                <button onClick={() => openEdit(item)}>Edit</button>\n")
	sb.WriteString("                <button onClick={() => handleDelete(item.id)}>Del</button>\n")
	sb.WriteString("              </td>\n")
	sb.WriteString("            </tr>\n")
	sb.WriteString("          ))}\n")
	sb.WriteString("        </tbody>\n")
	sb.WriteString("      </table>\n")
	sb.WriteString("    </div>\n")
	sb.WriteString("  );\n")
	sb.WriteString("}\n")

	return sb.String()
}

// GenerateMain returns Go source code for the generated web app's main.go.
// It wires a GORM PostgreSQL connection and the Gin router together.
// appImport is the module path of the generated app (e.g. "attendance-journal").
func GenerateMain(cfg *Config, appImport string) string {
	models := make([]string, len(cfg.Models))
	for i, m := range cfg.Models {
		models[i] = toPascalCase(toSingular(m.Name))
	}

	data := struct {
		ModelsImport string
		RoutesImport string
		DBHost       string
		DBName       string
		Port         int
		Models       []string
	}{
		ModelsImport: fmt.Sprintf("%q", appImport+"/models"),
		RoutesImport: fmt.Sprintf("%q", appImport+"/routes"),
		DBHost:       fmt.Sprintf("%q", cfg.Database.Host),
		DBName:       fmt.Sprintf("%q", cfg.Database.Name),
		Port:         cfg.App.Port,
		Models:       models,
	}

	var buf strings.Builder
	template.Must(template.New("main").Parse(mainTmpl)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}

// GenerateDockerCompose returns a docker-compose.yml for the generated app,
// wiring the Go server and a PostgreSQL service together.
func GenerateDockerCompose(cfg *Config) string {
	data := struct {
		Port   int
		DBName string
	}{
		Port:   cfg.App.Port,
		DBName: cfg.Database.Name,
	}
	var buf strings.Builder
	template.Must(template.New("docker-compose").Parse(dockerComposeTmpl)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}

// GenerateGoMod returns go.mod content for the generated app with
// gin, gorm, and the postgres driver as dependencies.
func GenerateGoMod(cfg *Config) string {
	data := struct{ ModuleName string }{ModuleName: cfg.App.Name}
	var buf strings.Builder
	template.Must(template.New("go.mod").Parse(goModTmpl)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}

// GenerateEnv returns a .env file with database connection variables.
func GenerateEnv(cfg *Config) string {
	data := struct {
		DBHost     string
		DBPort     int
		DBUser     string
		DBPassword string
		DBName     string
	}{
		DBHost:     cfg.Database.Host,
		DBPort:     cfg.Database.Port,
		DBUser:     cfg.Database.User,
		DBPassword: cfg.Database.Password,
		DBName:     cfg.Database.Name,
	}
	var buf strings.Builder
	template.Must(template.New(".env").Parse(envTmpl)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}

// GenerateDevScript returns a dev.sh that starts postgres, applies migrations,
// and runs the Go server.
func GenerateDevScript(cfg *Config) string {
	data := struct {
		DBUser     string
		DBPassword string
		DBName     string
		DBPort     int
		Port       int
	}{
		DBUser:     cfg.Database.User,
		DBPassword: cfg.Database.Password,
		DBName:     cfg.Database.Name,
		DBPort:     cfg.Database.Port,
		Port:       cfg.App.Port,
	}
	var buf strings.Builder
	template.Must(template.New("dev.sh").Parse(devShTmpl)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}

// GenerateShutdownScript returns a shutdown.sh that stops docker containers.
func GenerateShutdownScript() string {
	var buf strings.Builder
	template.Must(template.New("shutdown.sh").Parse(shutdownShTmpl)).Execute(&buf, nil) //nolint:errcheck
	return buf.String()
}

// GenerateGinRoutes returns Go source code with Gin CRUD handlers and a RegisterRoutes
// function for every model. modelsImport is the full import path of the models package
// (e.g. "myapp/models").
func GenerateGinRoutes(models []Model, pkgName string, modelsImport string) string {
	modPkg := modelsImport
	if idx := strings.LastIndex(modelsImport, "/"); idx >= 0 {
		modPkg = modelsImport[idx+1:]
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "package %s\n\n", pkgName)
	sb.WriteString("import (\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString("\t\"strconv\"\n\n")
	sb.WriteString("\t\"github.com/gin-gonic/gin\"\n")
	sb.WriteString("\t\"gorm.io/gorm\"\n")
	fmt.Fprintf(&sb, "\t%q\n", modelsImport)
	sb.WriteString(")\n\n")

	sb.WriteString("// RegisterRoutes wires all CRUD routes onto r.\n")
	sb.WriteString("func RegisterRoutes(r *gin.Engine, db *gorm.DB) {\n")
	for _, m := range models {
		base := "/" + m.Name
		s := toPascalCase(toSingular(m.Name))
		fmt.Fprintf(&sb, "\tr.GET(%q, list%s(db))\n", base, s)
		fmt.Fprintf(&sb, "\tr.GET(%q, get%s(db))\n", base+"/:id", s)
		fmt.Fprintf(&sb, "\tr.POST(%q, create%s(db))\n", base, s)
		fmt.Fprintf(&sb, "\tr.PUT(%q, update%s(db))\n", base+"/:id", s)
		fmt.Fprintf(&sb, "\tr.DELETE(%q, delete%s(db))\n", base+"/:id", s)
	}
	sb.WriteString("}\n")

	for _, m := range models {
		s := toPascalCase(toSingular(m.Name))
		typ := modPkg + "." + s

		fmt.Fprintf(&sb, `
func list%[1]s(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rows []%[2]s
		if err := db.Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}

func get%[1]s(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var row %[2]s
		if err := db.First(&row, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, row)
	}
}

func create%[1]s(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var row %[2]s
		if err := c.ShouldBindJSON(&row); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := db.Create(&row).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, row)
	}
}

func update%[1]s(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var row %[2]s
		if err := db.First(&row, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if err := c.ShouldBindJSON(&row); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		db.Save(&row)
		c.JSON(http.StatusOK, row)
	}
}

func delete%[1]s(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := db.Delete(&%[2]s{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
`, s, typ)
	}

	return sb.String()
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

	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.User == "" {
		cfg.Database.User = "postgres"
	}
	if cfg.Database.Password == "" {
		cfg.Database.Password = "secret"
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

	if err := os.MkdirAll("migrations", 0755); err != nil {
		log.Fatalf("create migrations dir: %v", err)
	}

	upSQL := GenerateMigrationUp(cfg.Models)
	if err := os.WriteFile("migrations/001_initial.up.sql", []byte(upSQL), 0644); err != nil {
		log.Fatalf("write up migration: %v", err)
	}
	fmt.Println("Generated migrations/001_initial.up.sql")

	downSQL := GenerateMigrationDown(cfg.Models)
	if err := os.WriteFile("migrations/001_initial.down.sql", []byte(downSQL), 0644); err != nil {
		log.Fatalf("write down migration: %v", err)
	}
	fmt.Println("Generated migrations/001_initial.down.sql")

	if err := os.MkdirAll("models", 0755); err != nil {
		log.Fatalf("create models dir: %v", err)
	}
	gormModels := GenerateGORMModels(cfg.Models, "models")
	if err := os.WriteFile("models/models.go", []byte(gormModels), 0644); err != nil {
		log.Fatalf("write models/models.go: %v", err)
	}
	fmt.Println("Generated models/models.go")

	if err := os.MkdirAll("routes", 0755); err != nil {
		log.Fatalf("create routes dir: %v", err)
	}
	ginRoutes := GenerateGinRoutes(cfg.Models, "routes", cfg.App.Name+"/models")
	if err := os.WriteFile("routes/routes.go", []byte(ginRoutes), 0644); err != nil {
		log.Fatalf("write routes/routes.go: %v", err)
	}
	fmt.Println("Generated routes/routes.go")

	mainSrc := GenerateMain(cfg, cfg.App.Name)
	if err := os.WriteFile("main.go", []byte(mainSrc), 0644); err != nil {
		log.Fatalf("write main.go: %v", err)
	}
	fmt.Println("Generated main.go")

	compose := GenerateDockerCompose(cfg)
	if err := os.WriteFile("docker-compose.yml", []byte(compose), 0644); err != nil {
		log.Fatalf("write docker-compose.yml: %v", err)
	}
	fmt.Println("Generated docker-compose.yml")

	goMod := GenerateGoMod(cfg)
	if err := os.WriteFile("go.mod", []byte(goMod), 0644); err != nil {
		log.Fatalf("write go.mod: %v", err)
	}
	fmt.Println("Generated go.mod")

	env := GenerateEnv(cfg)
	if err := os.WriteFile(".env", []byte(env), 0644); err != nil {
		log.Fatalf("write .env: %v", err)
	}
	fmt.Println("Generated .env")

	devSh := GenerateDevScript(cfg)
	if err := os.WriteFile("dev.sh", []byte(devSh), 0755); err != nil {
		log.Fatalf("write dev.sh: %v", err)
	}
	fmt.Println("Generated dev.sh")

	shutdownSh := GenerateShutdownScript()
	if err := os.WriteFile("shutdown.sh", []byte(shutdownSh), 0755); err != nil {
		log.Fatalf("write shutdown.sh: %v", err)
	}
	fmt.Println("Generated shutdown.sh")
}
