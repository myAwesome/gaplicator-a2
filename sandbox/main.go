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
func buildGORMTag(f Field) string {
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
	return `gorm:"` + strings.Join(parts, ";") + `"`
}

// GenerateGORMModels returns Go source code with GORM model structs for all models.
func GenerateGORMModels(models []Model, pkgName string) string {
	// Check if time.Time is needed for any field.
	needsTime := false
	for _, m := range models {
		for _, f := range m.Fields {
			if sqlTypeToGo(f.Type) == "time.Time" {
				needsTime = true
			}
		}
	}

	// Build struct name lookup: table name → struct name.
	structNames := make(map[string]string, len(models))
	for _, m := range models {
		structNames[m.Name] = toPascalCase(toSingular(m.Name))
	}

	var sb strings.Builder
	sb.WriteString("package " + pkgName + "\n\n")
	sb.WriteString("import (\n")
	if needsTime {
		sb.WriteString("\t\"time\"\n\n")
	}
	sb.WriteString("\t\"gorm.io/gorm\"\n")
	sb.WriteString(")\n")

	for _, m := range models {
		structName := structNames[m.Name]
		sb.WriteString("\ntype " + structName + " struct {\n")
		sb.WriteString("\tgorm.Model\n")

		for _, f := range m.Fields {
			fieldName := toPascalCase(f.Name)
			goType := sqlTypeToGo(f.Type)
			tag := buildGORMTag(f)
			fmt.Fprintf(&sb, "\t%-20s %-12s `%s`\n", fieldName, goType, tag)

			// For FK fields, add a typed association field.
			if f.References != "" {
				parts := strings.SplitN(f.References, ".", 2)
				if assocStruct, ok := structNames[parts[0]]; ok {
					// Strip "ID" or "Id" suffix to derive association field name.
					assocField := strings.TrimSuffix(fieldName, "ID")
					if assocField == fieldName {
						assocField = strings.TrimSuffix(fieldName, "Id")
					}
					if assocField == fieldName {
						assocField = assocStruct
					}
					fmt.Fprintf(&sb, "\t%-20s %-12s `gorm:\"foreignKey:%s\"`\n", assocField, assocStruct, fieldName)
				}
			}
		}

		sb.WriteString("}\n")
	}

	return sb.String()
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
}
