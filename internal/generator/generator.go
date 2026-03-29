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

//go:embed templates/migration_postgres.sql.tmpl
var migrationPostgresTmpl string

//go:embed templates/migration_mysql.sql.tmpl
var migrationMySQLTmpl string

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

//go:embed templates/auth.go.tmpl
var authGoTmpl string

//go:embed templates/readme.md.tmpl
var readmeTmpl string

type AuthConfig struct {
	Model string `yaml:"model"`
}

type Config struct {
	App      AppConfig      `yaml:"app"`
	Database DatabaseConfig `yaml:"database"`
	Auth     *AuthConfig    `yaml:"auth,omitempty"`
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
	Driver   string `yaml:"driver"` // "postgres" (default) or "mysql"
}

type Model struct {
	Name       string   `yaml:"name"`
	Timestamps *bool    `yaml:"timestamps,omitempty"`
	Fields     []Field  `yaml:"fields"`
	ManyToMany []string `yaml:"many_to_many"`
}

func modelHasTimestamps(m Model) bool {
	return m.Timestamps != nil && *m.Timestamps
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

	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "postgres"
	}
	if cfg.Database.Port == 0 {
		if cfg.Database.Driver == "mysql" {
			cfg.Database.Port = 3306
		} else {
			cfg.Database.Port = 5432
		}
	}
	if cfg.Database.User == "" {
		if cfg.Database.Driver == "mysql" {
			cfg.Database.User = "root"
		} else {
			cfg.Database.User = "postgres"
		}
	}
	if cfg.Database.Password == "" {
		cfg.Database.Password = "secret"
	}

	// If auth is enabled and the referenced model doesn't exist, create a default one.
	if cfg.Auth != nil && cfg.Auth.Model != "" {
		found := false
		for _, m := range cfg.Models {
			if m.Name == cfg.Auth.Model {
				found = true
				break
			}
		}
		if !found {
			cfg.Models = append(cfg.Models, Model{
				Name: cfg.Auth.Model,
				Fields: []Field{
					{Name: "email", Type: "varchar(255)", Required: true, Unique: true, Label: "Email"},
					{Name: "password", Type: "varchar(255)", Required: true, Label: "Password"},
					{Name: "name", Type: "varchar(100)", Label: "Name"},
				},
			})
		}
	}

	return &cfg, nil
}

// detectIdentityField returns the DB column name used to identify a user for login.
// Priority: first field named "email", then "username", then first varchar/text field.
func detectIdentityField(m Model) string {
	for _, f := range m.Fields {
		if f.Name == "email" || f.Name == "username" {
			return f.Name
		}
	}
	for _, f := range m.Fields {
		lower := strings.ToLower(f.Type)
		if strings.HasPrefix(lower, "varchar") || strings.HasPrefix(lower, "char") || lower == "text" {
			return f.Name
		}
	}
	if len(m.Fields) > 0 {
		return m.Fields[0].Name
	}
	return "id"
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
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "postgres"
	} else if cfg.Database.Driver != "postgres" && cfg.Database.Driver != "mysql" {
		errs = append(errs, fmt.Errorf("database.driver must be \"postgres\" or \"mysql\""))
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

		reservedFieldNames := map[string]bool{"id": true, "created_at": true, "updated_at": true, "deleted_at": true}
		for fi, f := range m.Fields {
			fprefix := fmt.Sprintf("%s field[%d]", prefix, fi)
			if f.Name != "" {
				fprefix = fmt.Sprintf("%s field %q", prefix, f.Name)
			}

			if f.Name == "" {
				errs = append(errs, fmt.Errorf("%s: name is required", fprefix))
			} else if !validIdentRe.MatchString(f.Name) {
				errs = append(errs, fmt.Errorf("%s: name must be lowercase snake_case (a-z, 0-9, _)", fprefix))
			} else if reservedFieldNames[f.Name] {
				errs = append(errs, fmt.Errorf("%s: field name %q is reserved and auto-generated", fprefix, f.Name))
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

	if cfg.Auth != nil {
		if cfg.Auth.Model == "" {
			errs = append(errs, fmt.Errorf("auth.model is required when auth is set"))
		} else if modelNameCount[cfg.Auth.Model] == 0 {
			errs = append(errs, fmt.Errorf("auth.model %q does not reference an existing model", cfg.Auth.Model))
		} else {
			hasPassword := false
			for _, m := range cfg.Models {
				if m.Name == cfg.Auth.Model {
					for _, f := range m.Fields {
						if strings.ToLower(f.Name) == "password" {
							hasPassword = true
							break
						}
					}
					break
				}
			}
			if !hasPassword {
				errs = append(errs, fmt.Errorf("auth.model %q must have a field named 'password'", cfg.Auth.Model))
			}
		}
	}

	return errs
}

type migrationField struct {
	Name       string
	SQLType    string
	Required   bool
	Unique     bool
	HasDefault bool
	Default    string
	HasRef     bool
	RefTable   string
	RefCol     string
}

type migrationIndex struct {
	IndexName string
	Table     string
	Column    string
}

type migrationModel struct {
	Name          string
	HasTimestamps bool
	Fields        []migrationField
	Indexes       []migrationIndex
}

type migrationData struct {
	Models     []migrationModel
	JoinTables []joinTableDef
}

func buildMigrationData(models []Model, driver string) migrationData {
	sorted := topoSort(models)
	tableModels := make([]migrationModel, len(sorted))
	for i, m := range sorted {
		var migFields []migrationField
		var indexes []migrationIndex
		for _, f := range m.Fields {
			mf := migrationField{
				Name:     f.Name,
				SQLType:  fieldSQLType(f, driver),
				Required: f.Required,
				Unique:   f.Unique,
			}
			if f.Default != nil {
				mf.HasDefault = true
				mf.Default = formatDefault(f.Default)
			}
			if f.References != "" {
				parts := strings.SplitN(f.References, ".", 2)
				mf.HasRef = true
				mf.RefTable = parts[0]
				mf.RefCol = parts[1]
			}
			migFields = append(migFields, mf)
			if f.Index && !f.Unique {
				indexes = append(indexes, migrationIndex{
					IndexName: fmt.Sprintf("idx_%s_%s", m.Name, f.Name),
					Table:     m.Name,
					Column:    f.Name,
				})
			}
		}
		tableModels[i] = migrationModel{
			Name:          m.Name,
			HasTimestamps: modelHasTimestamps(m),
			Fields:        migFields,
			Indexes:       indexes,
		}
	}
	return migrationData{Models: tableModels, JoinTables: collectJoinTables(models)}
}

func GenerateMigrationUp(models []Model, driver string) string {
	data := buildMigrationData(models, driver)
	tmplSrc := migrationPostgresTmpl
	if driver == "mysql" {
		tmplSrc = migrationMySQLTmpl
	}
	var buf strings.Builder
	template.Must(template.New("migration").Parse(tmplSrc)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}

func fieldSQLType(f Field, driver string) string {
	lower := strings.ToLower(f.Type)
	if lower == "enum" {
		quotedValues := make([]string, len(f.Values))
		for i, v := range f.Values {
			quotedValues[i] = "'" + strings.ReplaceAll(v, "'", "''") + "'"
		}
		if driver == "mysql" {
			return fmt.Sprintf("ENUM(%s)", strings.Join(quotedValues, ", "))
		}
		return fmt.Sprintf("TEXT CHECK (%s IN (%s))", f.Name, strings.Join(quotedValues, ", "))
	}
	if driver == "mysql" && lower == "uuid" {
		return "VARCHAR(36)"
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

func buildFieldTags(f Field, hideJSON bool) string {
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
	if hideJSON {
		return `gorm:"` + strings.Join(parts, ";") + `" json:"-"`
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
	StructName    string
	TableName     string
	HasTimestamps bool
	Fields        []gormFieldData
	M2MFields     []gormM2MField
}

func GenerateGORMModels(models []Model, pkgName string, auth *AuthConfig) string {
	structNames := make(map[string]string, len(models))
	for _, m := range models {
		structNames[m.Name] = toPascalCase(toSingular(m.Name))
	}

	modelData := make([]gormModelData, 0, len(models))
	for _, m := range models {
		isAuthModel := auth != nil && m.Name == auth.Model
		structName := structNames[m.Name]
		fields := make([]gormFieldData, 0, len(m.Fields))
		for _, f := range m.Fields {
			fieldName := toPascalCase(f.Name)
			goType := sqlTypeToGo(f.Type)
			if f.References != "" && !f.Required {
				goType = "*" + goType
			}
			isPasswordField := isAuthModel && strings.ToLower(f.Name) == "password"
			fd := gormFieldData{
				FieldName: fieldName,
				GoType:    goType,
				Tags:      buildFieldTags(f, isPasswordField),
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
		modelData = append(modelData, gormModelData{
			StructName:    structName,
			TableName:     m.Name,
			HasTimestamps: modelHasTimestamps(m),
			Fields:        fields,
			M2MFields:     m2mFields,
		})
	}

	hasAnyTimestamps := false
	needsTimeImport := false
	for _, m := range models {
		if modelHasTimestamps(m) {
			hasAnyTimestamps = true
			needsTimeImport = true
		}
		if !needsTimeImport {
			for _, f := range m.Fields {
				if sqlTypeToGo(f.Type) == "time.Time" {
					needsTimeImport = true
					break
				}
			}
		}
	}

	data := struct {
		PkgName          string
		Models           []gormModelData
		HasAnyTimestamps bool
		NeedsTimeImport  bool
	}{PkgName: pkgName, Models: modelData, HasAnyTimestamps: hasAnyTimestamps, NeedsTimeImport: needsTimeImport}

	var buf strings.Builder
	template.Must(template.New("gorm_models").Parse(gormModelsTmpl)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}

func GenerateMain(cfg *Config, appImport string) (string, error) {
	data := struct {
		RoutesImport string
		DBHost       string
		DBName       string
		DBPort       int
		Port         int
		HasAuth      bool
		IsMySQL      bool
	}{
		RoutesImport: fmt.Sprintf("%q", appImport+"/routes"),
		DBHost:       fmt.Sprintf("%q", cfg.Database.Host),
		DBName:       fmt.Sprintf("%q", cfg.Database.Name),
		DBPort:       cfg.Database.Port,
		Port:         cfg.App.Port,
		HasAuth:      cfg.Auth != nil,
		IsMySQL:      cfg.Database.Driver == "mysql",
	}

	var buf strings.Builder
	if err := template.Must(template.New("main").Parse(mainTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute main template: %w", err)
	}
	return buf.String(), nil
}

func GenerateDockerCompose(cfg *Config) (string, error) {
	data := struct {
		Port       int
		DBPort     int
		DBName     string
		DBUser     string
		DBPassword string
		IsMySQL    bool
	}{
		Port:       cfg.App.Port,
		DBPort:     cfg.Database.Port,
		DBName:     cfg.Database.Name,
		DBUser:     cfg.Database.User,
		DBPassword: cfg.Database.Password,
		IsMySQL:    cfg.Database.Driver == "mysql",
	}
	var buf strings.Builder
	if err := template.Must(template.New("docker-compose").Parse(dockerComposeTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute docker-compose template: %w", err)
	}
	return buf.String(), nil
}

func GenerateGoMod(cfg *Config) (string, error) {
	data := struct {
		ModuleName string
		HasAuth    bool
		IsMySQL    bool
	}{ModuleName: cfg.App.Name, HasAuth: cfg.Auth != nil, IsMySQL: cfg.Database.Driver == "mysql"}
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
		HasAuth    bool
		IsMySQL    bool
	}{
		DBHost:     cfg.Database.Host,
		DBPort:     cfg.Database.Port,
		DBUser:     cfg.Database.User,
		DBPassword: cfg.Database.Password,
		DBName:     cfg.Database.Name,
		HasAuth:    cfg.Auth != nil,
		IsMySQL:    cfg.Database.Driver == "mysql",
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
		IsMySQL    bool
	}{
		DBUser:     cfg.Database.User,
		DBPassword: cfg.Database.Password,
		DBName:     cfg.Database.Name,
		DBPort:     cfg.Database.Port,
		Port:       cfg.App.Port,
		IsMySQL:    cfg.Database.Driver == "mysql",
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

func GenerateReadme(cfg *Config) (string, error) {
	identityField := "email"
	if cfg.Auth != nil {
		for _, m := range cfg.Models {
			if m.Name == cfg.Auth.Model {
				identityField = detectIdentityField(m)
				break
			}
		}
	}
	data := struct {
		AppName       string
		Port          int
		IsMySQL       bool
		HasAuth       bool
		IdentityField string
		Models        []Model
	}{
		AppName:       cfg.App.Name,
		Port:          cfg.App.Port,
		IsMySQL:       cfg.Database.Driver == "mysql",
		HasAuth:       cfg.Auth != nil,
		IdentityField: identityField,
		Models:        cfg.Models,
	}
	var buf strings.Builder
	if err := template.Must(template.New("readme.md").Parse(readmeTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute readme.md template: %w", err)
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

func GenerateGinRoutes(models []Model, pkgName string, modelsImport string, isMySQL bool) string {
	modPkg := modelsImport
	if idx := strings.LastIndex(modelsImport, "/"); idx >= 0 {
		modPkg = modelsImport[idx+1:]
	}

	hasAnySearch := false
	hasAnyM2M := false
	ginModels := make([]ginModelData, 0, len(models))
	for _, m := range models {
		s := toPascalCase(toSingular(m.Name))
		sortCols := []string{"id"}
		if modelHasTimestamps(m) {
			sortCols = append(sortCols, "created_at", "updated_at")
		}
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

	likeOp := "ILIKE"
	if isMySQL {
		likeOp = "LIKE"
	}

	data := struct {
		PkgName      string
		ModelsImport string
		HasSearch    bool
		HasAnyM2M    bool
		Models       []ginModelData
		LikeOp       string
	}{PkgName: pkgName, ModelsImport: modelsImport, HasSearch: hasAnySearch, HasAnyM2M: hasAnyM2M, Models: ginModels, LikeOp: likeOp}

	var buf strings.Builder
	template.Must(template.New("gin_routes").Parse(ginRoutesTmpl)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}

type authTemplateData struct {
	ModelsImport   string
	AuthStructName string
	IdentityField  string
	IdentityGoName string
	PasswordGoName string
}

func GenerateAuthGo(cfg *Config, appImport string) (string, error) {
	if cfg.Auth == nil {
		return "", fmt.Errorf("auth config is not set")
	}
	var authModel Model
	for _, m := range cfg.Models {
		if m.Name == cfg.Auth.Model {
			authModel = m
			break
		}
	}
	identityField := detectIdentityField(authModel)
	data := authTemplateData{
		ModelsImport:   fmt.Sprintf("%q", appImport+"/models"),
		AuthStructName: toPascalCase(toSingular(authModel.Name)),
		IdentityField:  identityField,
		IdentityGoName: toPascalCase(identityField),
		PasswordGoName: "Password",
	}
	var buf strings.Builder
	if err := template.Must(template.New("auth.go").Parse(authGoTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute auth.go template: %w", err)
	}
	return buf.String(), nil
}
