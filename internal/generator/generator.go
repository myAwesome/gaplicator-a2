package generator

import (
	_ "embed"
	"fmt"
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

//go:embed templates/gorm_models.go.tmpl
var gormModelsTmpl string

//go:embed templates/gin_routes.go.tmpl
var ginRoutesTmpl string

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
	References string `yaml:"references"`
}

var validIdentRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

var validTypeRe = regexp.MustCompile(
	`^(int|bigint|smallint|text|boolean|bool|date|datetime|timestamp|uuid|float|double|` +
		`varchar\(\d+\)|char\(\d+\)|decimal\(\d+,\s*\d+\))$`,
)

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

func ValidateConfig(cfg *Config) []error {
	var errs []error

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
		} else if !validIdentRe.MatchString(m.Name) {
			errs = append(errs, fmt.Errorf("model %q: name must be lowercase snake_case (a-z, 0-9, _)", m.Name))
			prefix = fmt.Sprintf("model %q", m.Name)
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
			} else if !validIdentRe.MatchString(f.Name) {
				errs = append(errs, fmt.Errorf("%s: name must be lowercase snake_case (a-z, 0-9, _)", fprefix))
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
				} else if !validIdentRe.MatchString(parts[0]) || !validIdentRe.MatchString(parts[1]) {
					errs = append(errs, fmt.Errorf("%s: references %q identifiers must be lowercase snake_case", fprefix, f.References))
				} else if !modelNames[parts[0]] {
					errs = append(errs, fmt.Errorf("%s: references unknown model %q", fprefix, parts[0]))
				}
			}
		}
	}

	return errs
}

func GenerateMigrationUp(models []Model) string {
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
	fmt.Fprintf(&sb, "CREATE TABLE IF NOT EXISTS %s (\n", m.Name)
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
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	default:
		return fmt.Sprintf("%v", val)
	}
}

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

var goAcronyms = map[string]string{
	"id": "ID", "url": "URL", "uri": "URI", "api": "API",
	"http": "HTTP", "https": "HTTPS", "sql": "SQL", "db": "DB",
	"uuid": "UUID", "ip": "IP",
}

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

type gormFieldData struct {
	FieldName  string
	GoType     string
	Tags       string
	AssocField string
	AssocType  string
	AssocTags  string
}

type gormModelData struct {
	StructName string
	Fields     []gormFieldData
}

func GenerateGORMModels(models []Model, pkgName string) string {
	structNames := make(map[string]string, len(models))
	for _, m := range models {
		structNames[m.Name] = toPascalCase(toSingular(m.Name))
	}

	modelData := make([]gormModelData, 0, len(models))
	for _, m := range models {
		structName := structNames[m.Name]
		fields := make([]gormFieldData, 0, len(m.Fields))
		for _, f := range m.Fields {
			fieldName := toPascalCase(f.Name)
			fd := gormFieldData{
				FieldName: fieldName,
				GoType:    sqlTypeToGo(f.Type),
				Tags:      buildFieldTags(f),
			}
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
					fd.AssocField = assocField
					fd.AssocType = assocStruct
					fd.AssocTags = fmt.Sprintf("gorm:\"foreignKey:%s\" json:\"%s\"", fieldName, assocJsonName)
				}
			}
			fields = append(fields, fd)
		}
		modelData = append(modelData, gormModelData{StructName: structName, Fields: fields})
	}

	data := struct {
		PkgName string
		Models  []gormModelData
	}{PkgName: pkgName, Models: modelData}

	var buf strings.Builder
	template.Must(template.New("gorm_models").Parse(gormModelsTmpl)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}

func GenerateMain(cfg *Config, appImport string) (string, error) {
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
	if err := template.Must(template.New("main").Parse(mainTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute main template: %w", err)
	}
	return buf.String(), nil
}

func GenerateDockerCompose(cfg *Config) (string, error) {
	data := struct {
		Port   int
		DBName string
	}{
		Port:   cfg.App.Port,
		DBName: cfg.Database.Name,
	}
	var buf strings.Builder
	if err := template.Must(template.New("docker-compose").Parse(dockerComposeTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute docker-compose template: %w", err)
	}
	return buf.String(), nil
}

func GenerateGoMod(cfg *Config) (string, error) {
	data := struct{ ModuleName string }{ModuleName: cfg.App.Name}
	var buf strings.Builder
	if err := template.Must(template.New("go.mod").Parse(goModTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute go.mod template: %w", err)
	}
	return buf.String(), nil
}

func GenerateEnv(cfg *Config) (string, error) {
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
	if err := template.Must(template.New(".env").Parse(envTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute .env template: %w", err)
	}
	return buf.String(), nil
}

func GenerateDevScript(cfg *Config) (string, error) {
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
	if err := template.Must(template.New("dev.sh").Parse(devShTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute dev.sh template: %w", err)
	}
	return buf.String(), nil
}

func GenerateShutdownScript() (string, error) {
	var buf strings.Builder
	if err := template.Must(template.New("shutdown.sh").Parse(shutdownShTmpl)).Execute(&buf, nil); err != nil {
		return "", fmt.Errorf("execute shutdown.sh template: %w", err)
	}
	return buf.String(), nil
}

type ginModelData struct {
	Name     string
	Singular string
	Type     string
	Base     string
	BaseID   string
}

func GenerateGinRoutes(models []Model, pkgName string, modelsImport string) string {
	modPkg := modelsImport
	if idx := strings.LastIndex(modelsImport, "/"); idx >= 0 {
		modPkg = modelsImport[idx+1:]
	}

	ginModels := make([]ginModelData, 0, len(models))
	for _, m := range models {
		s := toPascalCase(toSingular(m.Name))
		ginModels = append(ginModels, ginModelData{
			Name:     m.Name,
			Singular: s,
			Type:     modPkg + "." + s,
			Base:     "/" + m.Name,
			BaseID:   "/" + m.Name + "/:id",
		})
	}

	data := struct {
		PkgName      string
		ModelsImport string
		Models       []ginModelData
	}{PkgName: pkgName, ModelsImport: modelsImport, Models: ginModels}

	var buf strings.Builder
	template.Must(template.New("gin_routes").Parse(ginRoutesTmpl)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}
