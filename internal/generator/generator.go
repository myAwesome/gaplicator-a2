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
	Name       string   `yaml:"name"`
	Fields     []Field  `yaml:"fields"`
	ManyToMany []string `yaml:"many_to_many"`
}

type Field struct {
	Name         string   `yaml:"name"`
	Type         string   `yaml:"type"`
	Values       []string `yaml:"values"`
	Required     bool     `yaml:"required"`
	Unique       bool     `yaml:"unique"`
	Default      any      `yaml:"default"`
	References   string   `yaml:"references"`
	DisplayField string   `yaml:"display_field"`
	Index        bool     `yaml:"index"`
	Label        string   `yaml:"label"`
}

var validIdentRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

var validAppNameRe = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

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
	} else if !validAppNameRe.MatchString(cfg.App.Name) {
		errs = append(errs, fmt.Errorf("app.name must be lowercase letters, digits, hyphens, or underscores"))
	}
	if cfg.App.Port < 1 || cfg.App.Port > 65535 {
		errs = append(errs, fmt.Errorf("app.port must be between 1 and 65535"))
	}
	if cfg.Database.Host == "" {
		errs = append(errs, fmt.Errorf("database.host is required"))
	}
	if cfg.Database.Name == "" {
		errs = append(errs, fmt.Errorf("database.name is required"))
	}
	if cfg.Database.Port < 1 || cfg.Database.Port > 65535 {
		errs = append(errs, fmt.Errorf("database.port must be between 1 and 65535"))
	}
	if len(cfg.Models) == 0 {
		errs = append(errs, fmt.Errorf("at least one model is required"))
	}

	// Count model names to detect duplicates.
	modelNameCount := make(map[string]int, len(cfg.Models))
	for _, m := range cfg.Models {
		if m.Name != "" {
			modelNameCount[m.Name]++
		}
	}
	for name, count := range modelNameCount {
		if count > 1 {
			errs = append(errs, fmt.Errorf("duplicate model name %q", name))
		}
	}

	// Build a field registry per model for reference validation.
	modelFields := make(map[string]map[string]bool, len(cfg.Models))
	for _, m := range cfg.Models {
		if m.Name == "" {
			continue
		}
		fields := make(map[string]bool, len(m.Fields)+1)
		fields["id"] = true // auto-generated primary key
		for _, f := range m.Fields {
			if f.Name != "" {
				fields[f.Name] = true
			}
		}
		modelFields[m.Name] = fields
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

		for _, other := range m.ManyToMany {
			if other == "" {
				errs = append(errs, fmt.Errorf("%s: many_to_many entry must not be empty", prefix))
			} else if !validIdentRe.MatchString(other) {
				errs = append(errs, fmt.Errorf("%s: many_to_many entry %q must be lowercase snake_case", prefix, other))
			} else if other == m.Name {
				errs = append(errs, fmt.Errorf("%s: many_to_many cannot reference itself", prefix))
			} else if modelNameCount[other] == 0 {
				errs = append(errs, fmt.Errorf("%s: many_to_many references unknown model %q", prefix, other))
			}
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
			} else if strings.ToLower(f.Type) == "enum" {
				if len(f.Values) == 0 {
					errs = append(errs, fmt.Errorf("%s: enum type requires at least one value in 'values'", fprefix))
				}
			} else if !validTypeRe.MatchString(strings.ToLower(f.Type)) {
				errs = append(errs, fmt.Errorf("%s: unknown type %q", fprefix, f.Type))
			}

			if f.References != "" {
				parts := strings.SplitN(f.References, ".", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					errs = append(errs, fmt.Errorf("%s: references %q must be in \"model.field\" format", fprefix, f.References))
				} else if !validIdentRe.MatchString(parts[0]) || !validIdentRe.MatchString(parts[1]) {
					errs = append(errs, fmt.Errorf("%s: references %q identifiers must be lowercase snake_case", fprefix, f.References))
				} else if modelNameCount[parts[0]] == 0 {
					errs = append(errs, fmt.Errorf("%s: references unknown model %q", fprefix, parts[0]))
				} else if fields, ok := modelFields[parts[0]]; ok && !fields[parts[1]] {
					errs = append(errs, fmt.Errorf("%s: references unknown field %q in model %q", fprefix, parts[1], parts[0]))
				} else if f.DisplayField != "" {
					refModel := parts[0]
					if fields, ok := modelFields[refModel]; ok && !fields[f.DisplayField] {
						errs = append(errs, fmt.Errorf("%s: display_field %q does not exist in model %q", fprefix, f.DisplayField, refModel))
					}
				}
			} else if f.DisplayField != "" {
				errs = append(errs, fmt.Errorf("%s: display_field requires references to be set", fprefix))
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
	joins := collectJoinTables(models)
	for _, j := range joins {
		sb.WriteString("\n")
		sb.WriteString(joinTableSQL(j))
	}
	return sb.String()
}

func GenerateMigrationDown(models []Model) string {
	sorted := topoSort(models)
	var sb strings.Builder
	// Drop join tables first (they reference the main tables).
	joins := collectJoinTables(models)
	for i := len(joins) - 1; i >= 0; i-- {
		fmt.Fprintf(&sb, "DROP TABLE IF EXISTS %s;\n", joins[i].Table)
	}
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
		fmt.Fprintf(&sb, "    %s %s", f.Name, fieldSQLType(f))
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
	for _, f := range m.Fields {
		if f.Index && !f.Unique {
			fmt.Fprintf(&sb, "CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s (%s);\n", m.Name, f.Name, m.Name, f.Name)
		}
	}
	return sb.String()
}

func fieldSQLType(f Field) string {
	if strings.ToLower(f.Type) == "enum" {
		quotedValues := make([]string, len(f.Values))
		for i, v := range f.Values {
			quotedValues[i] = "'" + strings.ReplaceAll(v, "'", "''") + "'"
		}
		return fmt.Sprintf("TEXT CHECK (%s IN (%s))", f.Name, strings.Join(quotedValues, ", "))
	}
	return strings.ToUpper(f.Type)
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

// joinTableName returns the canonical join table name for models a and b
// (names sorted alphabetically and joined with "_").
func joinTableName(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "_" + b
}

type joinTableDef struct {
	Table string
	AName string
	BName string
	ACol  string
	BCol  string
}

// collectJoinTables returns deduplicated join table definitions for all M2M pairs.
func collectJoinTables(models []Model) []joinTableDef {
	seen := map[string]bool{}
	var result []joinTableDef
	for _, m := range models {
		for _, other := range m.ManyToMany {
			name := joinTableName(m.Name, other)
			if seen[name] {
				continue
			}
			seen[name] = true
			a, b := m.Name, other
			if a > b {
				a, b = b, a
			}
			result = append(result, joinTableDef{
				Table: name,
				AName: a,
				BName: b,
				ACol:  toSingular(a) + "_id",
				BCol:  toSingular(b) + "_id",
			})
		}
	}
	return result
}

func joinTableSQL(j joinTableDef) string {
	return fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (\n    %s INT NOT NULL REFERENCES %s(id) ON DELETE CASCADE,\n    %s INT NOT NULL REFERENCES %s(id) ON DELETE CASCADE,\n    PRIMARY KEY (%s, %s)\n);\n",
		j.Table,
		j.ACol, j.AName,
		j.BCol, j.BName,
		j.ACol, j.BCol,
	)
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
	case strings.HasPrefix(lower, "varchar"), strings.HasPrefix(lower, "char"), lower == "text", lower == "uuid", lower == "enum":
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
	} else if f.Index {
		parts = append(parts, "index")
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

type gormM2MField struct {
	FieldName string
	SliceType string
	Tags      string
}

type gormModelData struct {
	StructName string
	Fields     []gormFieldData
	M2MFields  []gormM2MField
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
			goType := sqlTypeToGo(f.Type)
			if f.References != "" && !f.Required {
				goType = "*" + goType
			}
			fd := gormFieldData{
				FieldName: fieldName,
				GoType:    goType,
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
		var m2mFields []gormM2MField
		for _, otherName := range m.ManyToMany {
			if otherStruct, ok := structNames[otherName]; ok {
				jt := joinTableName(m.Name, otherName)
				tags := fmt.Sprintf(`gorm:"many2many:%s;" json:"%s,omitempty"`, jt, otherName)
				m2mFields = append(m2mFields, gormM2MField{
					FieldName: toPascalCase(otherName),
					SliceType: "[]" + otherStruct,
					Tags:      tags,
				})
			}
		}
		modelData = append(modelData, gormModelData{StructName: structName, Fields: fields, M2MFields: m2mFields})
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

type ginFilterColumn struct {
	DBName    string
	IsNumeric bool
	IsBool    bool
}

type ginM2MRelation struct {
	AssocField string
	AssocType  string
	IDsField   string
}

type ginModelData struct {
	Name          string
	Singular      string
	Type          string
	Base          string
	BaseID        string
	BaseBatch     string
	SortColumns   []string
	SearchColumns []string
	FilterColumns []ginFilterColumn
	HasSearch     bool
	M2MRelations  []ginM2MRelation
	HasM2M        bool
}

func GenerateGinRoutes(models []Model, pkgName string, modelsImport string) string {
	modPkg := modelsImport
	if idx := strings.LastIndex(modelsImport, "/"); idx >= 0 {
		modPkg = modelsImport[idx+1:]
	}

	hasAnySearch := false
	hasAnyM2M := false
	ginModels := make([]ginModelData, 0, len(models))
	for _, m := range models {
		s := toPascalCase(toSingular(m.Name))
		sortCols := []string{"id", "created_at", "updated_at"}
		var searchCols []string
		var filterCols []ginFilterColumn
		for _, f := range m.Fields {
			sortCols = append(sortCols, f.Name)
			lower := strings.ToLower(f.Type)
			isText := strings.HasPrefix(lower, "text") || strings.HasPrefix(lower, "varchar") || strings.HasPrefix(lower, "char")
			if isText {
				searchCols = append(searchCols, f.Name)
			}
			isNum := lower == "int" || lower == "bigint" || lower == "smallint" ||
				lower == "float" || lower == "double" || strings.HasPrefix(lower, "decimal") ||
				f.References != ""
			isBool := lower == "boolean" || lower == "bool"
			filterCols = append(filterCols, ginFilterColumn{DBName: f.Name, IsNumeric: isNum, IsBool: isBool})
		}
		hasSearch := len(searchCols) > 0
		if hasSearch {
			hasAnySearch = true
		}
		var m2mRelations []ginM2MRelation
		for _, other := range m.ManyToMany {
			otherStruct := toPascalCase(toSingular(other))
			m2mRelations = append(m2mRelations, ginM2MRelation{
				AssocField: toPascalCase(other),
				AssocType:  modPkg + "." + otherStruct,
				IDsField:   toSingular(other) + "_ids",
			})
		}
		if len(m2mRelations) > 0 {
			hasAnyM2M = true
		}
		ginModels = append(ginModels, ginModelData{
			Name:          m.Name,
			Singular:      s,
			Type:          modPkg + "." + s,
			Base:          "/" + m.Name,
			BaseID:        "/" + m.Name + "/:id",
			BaseBatch:     "/" + m.Name + "/batch",
			SortColumns:   sortCols,
			SearchColumns: searchCols,
			FilterColumns: filterCols,
			HasSearch:     hasSearch,
			M2MRelations:  m2mRelations,
			HasM2M:        len(m2mRelations) > 0,
		})
	}

	data := struct {
		PkgName      string
		ModelsImport string
		HasSearch    bool
		HasAnyM2M    bool
		Models       []ginModelData
	}{PkgName: pkgName, ModelsImport: modelsImport, HasSearch: hasAnySearch, HasAnyM2M: hasAnyM2M, Models: ginModels}

	var buf strings.Builder
	template.Must(template.New("gin_routes").Parse(ginRoutesTmpl)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}
