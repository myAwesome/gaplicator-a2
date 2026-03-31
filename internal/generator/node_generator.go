package generator

import (
	"fmt"
	"strings"
	"text/template"
)

// ── package.json ──────────────────────────────────────────────────────────────

func GenerateNodePackageJSON(cfg *Config) (string, error) {
	data := struct {
		AppName string
		HasAuth bool
	}{AppName: cfg.App.Name, HasAuth: cfg.Auth != nil}
	var buf strings.Builder
	if err := template.Must(template.New("node_package_json").Parse(nodePackageJSONTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute node_package_json template: %w", err)
	}
	return buf.String(), nil
}

// ── index.js ──────────────────────────────────────────────────────────────────

func GenerateNodeIndex(cfg *Config) (string, error) {
	data := struct {
		Port    int
		HasAuth bool
	}{Port: cfg.App.Port, HasAuth: cfg.Auth != nil}
	var buf strings.Builder
	if err := template.Must(template.New("node_index_js").Parse(nodeIndexJSTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute node_index_js template: %w", err)
	}
	return buf.String(), nil
}

// ── routes.js ─────────────────────────────────────────────────────────────────

type nodeFilterColumn struct {
	Name      string
	IsNumeric bool
	IsBool    bool
}

type nodeM2MRelation struct {
	Field     string // camelCase plural  e.g. "tags"
	ModelName string // PascalCase        e.g. "Tag"
	IdsField  string // e.g. "tag_ids"
	JoinTable string
}

type nodeModelData struct {
	Name          string
	CamelName     string
	SortColumns   []string
	SearchColumns []string
	FilterColumns []nodeFilterColumn
	HasM2M        bool
	M2MRelations  []nodeM2MRelation
	HasPlainFields bool
}

type nodeRoutesData struct {
	Models []nodeModelData
	IsPg   bool
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 0 {
		return s
	}
	out := parts[0]
	for _, p := range parts[1:] {
		if p == "" {
			continue
		}
		out += strings.ToUpper(p[:1]) + p[1:]
	}
	return out
}

func buildNodeRouteData(models []Model, isMySQL bool) nodeRoutesData {
	nodeModels := make([]nodeModelData, 0, len(models))
	for _, m := range models {
		sortCols := []string{"id"}
		if modelHasTimestamps(m) {
			sortCols = append(sortCols, "createdAt", "updatedAt")
		}
		var searchCols []string
		var filterCols []nodeFilterColumn
		for _, f := range m.Fields {
			sortCols = append(sortCols, toCamelCase(f.Name))
			lower := strings.ToLower(f.Type)
			isText := strings.HasPrefix(lower, "text") || strings.HasPrefix(lower, "varchar") || strings.HasPrefix(lower, "char")
			if isText {
				searchCols = append(searchCols, toCamelCase(f.Name))
			}
			isNum := lower == "int" || lower == "bigint" || lower == "smallint" ||
				lower == "float" || lower == "double" || strings.HasPrefix(lower, "decimal") ||
				f.References != ""
			isBool := lower == "boolean" || lower == "bool"
			filterCols = append(filterCols, nodeFilterColumn{
				Name:      toCamelCase(f.Name),
				IsNumeric: isNum,
				IsBool:    isBool,
			})
		}
		var m2mRels []nodeM2MRelation
		for _, other := range m.ManyToMany {
			jt := joinTableName(m.Name, other)
			otherStruct := toPascalCase(toSingular(other))
			m2mRels = append(m2mRels, nodeM2MRelation{
				Field:     toCamelCase(other),
				ModelName: otherStruct,
				IdsField:  toSingular(other) + "_ids",
				JoinTable: jt,
			})
		}
		nodeModels = append(nodeModels, nodeModelData{
			Name:          m.Name,
			CamelName:     toCamelCase(toSingular(m.Name)),
			SortColumns:   sortCols,
			SearchColumns: searchCols,
			FilterColumns: filterCols,
			HasM2M:        len(m2mRels) > 0,
			M2MRelations:  m2mRels,
			HasPlainFields: len(m.Fields) > 0,
		})
	}
	return nodeRoutesData{Models: nodeModels, IsPg: !isMySQL}
}

func GenerateNodeRoutes(cfg *Config) (string, error) {
	data := buildNodeRouteData(cfg.Models, cfg.Database.Driver == "mysql")
	var buf strings.Builder
	if err := template.Must(template.New("node_routes_js").Parse(nodeRoutesJSTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute node_routes_js template: %w", err)
	}
	return buf.String(), nil
}

// ── auth.js ───────────────────────────────────────────────────────────────────

func GenerateNodeAuth(cfg *Config) (string, error) {
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
	data := struct {
		IdentityField string
		ModelCamel    string
	}{
		IdentityField: identityField,
		ModelCamel:    toCamelCase(toSingular(authModel.Name)),
	}
	var buf strings.Builder
	if err := template.Must(template.New("node_auth_js").Parse(nodeAuthJSTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute node_auth_js template: %w", err)
	}
	return buf.String(), nil
}

// ── prisma/schema.prisma ──────────────────────────────────────────────────────

type prismaFieldData struct {
	Name       string
	PrismaType string
	IsRequired bool
	Attrs      []string
}

type prismaM2MRelation struct {
	Field     string
	ModelName string
	JoinTable string
}

type prismaBackRelation struct {
	Field     string
	ModelName string
}

type prismaModelData struct {
	StructName    string
	TableName     string
	HasTimestamps bool
	Fields        []prismaFieldData
	M2MRelations  []prismaM2MRelation
	BackRelations []prismaBackRelation
}

type prismaSchemaData struct {
	DBProvider string
	Models     []prismaModelData
}

func sqlTypeToPrisma(f Field) (prismaType string, attrs []string) {
	lower := strings.ToLower(f.Type)
	switch {
	case strings.HasPrefix(lower, "varchar"), strings.HasPrefix(lower, "char"), lower == "text", lower == "enum":
		prismaType = "String"
	case lower == "int", lower == "smallint":
		prismaType = "Int"
	case lower == "bigint":
		prismaType = "BigInt"
	case lower == "boolean", lower == "bool":
		prismaType = "Boolean"
	case lower == "date":
		prismaType = "DateTime"
	case lower == "datetime", lower == "timestamp":
		prismaType = "DateTime"
	case lower == "float", lower == "double":
		prismaType = "Float"
	case strings.HasPrefix(lower, "decimal"):
		prismaType = "Decimal"
	case lower == "uuid":
		prismaType = "String"
		attrs = append(attrs, "@db.Uuid")
	default:
		prismaType = "String"
	}

	if f.References != "" {
		prismaType = "Int"
		if !f.Required {
			prismaType = "Int?"
		}
	}

	if f.Default != nil {
		attrs = append(attrs, fmt.Sprintf("@default(%v)", f.Default))
	}
	if f.Unique {
		attrs = append(attrs, "@unique")
	}
	if f.Index && !f.Unique {
		attrs = append(attrs, "// @index — add @@index(["+f.Name+"]) at model level if needed")
	}
	if f.References != "" {
		parts := strings.SplitN(f.References, ".", 2)
		attrs = append(attrs, fmt.Sprintf("@relation(fields: [%s], references: [%s])", f.Name, parts[1]))
	}
	return
}

func GenerateNodePrismaSchema(cfg *Config) (string, error) {
	dbProvider := "postgresql"
	if cfg.Database.Driver == "mysql" {
		dbProvider = "mysql"
	}

	// build a map of which models are referenced (back-relations)
	type backRef struct {
		fromModel string
		fieldName string
	}
	backRefs := make(map[string][]backRef) // key = referenced model name
	for _, m := range cfg.Models {
		for _, f := range m.Fields {
			if f.References != "" {
				parts := strings.SplitN(f.References, ".", 2)
				refModel := parts[0]
				backRefs[refModel] = append(backRefs[refModel], backRef{
					fromModel: m.Name,
					fieldName: toCamelCase(m.Name),
				})
			}
		}
	}

	prismaModels := make([]prismaModelData, 0, len(cfg.Models))
	for _, m := range cfg.Models {
		structName := toPascalCase(toSingular(m.Name))
		fields := make([]prismaFieldData, 0, len(m.Fields))
		for _, f := range m.Fields {
			pt, attrs := sqlTypeToPrisma(f)
			mapAttr := fmt.Sprintf("@map(\"%s\")", f.Name)
			allAttrs := append([]string{mapAttr}, attrs...)

			isReq := f.Required
			if f.References != "" {
				// The FK field itself (e.g. user_id Int?)
				fields = append(fields, prismaFieldData{
					Name:       toCamelCase(f.Name),
					PrismaType: pt,
					IsRequired: isReq,
					Attrs:      []string{mapAttr},
				})
				// The relation field (e.g. user User @relation(...))
				refParts := strings.SplitN(f.References, ".", 2)
				refStruct := toPascalCase(toSingular(refParts[0]))
				relType := refStruct
				if !f.Required {
					relType = refStruct + "?"
				}
				relAttr := fmt.Sprintf("@relation(fields: [%s], references: [%s])", toCamelCase(f.Name), refParts[1])
				assocName := strings.TrimSuffix(toCamelCase(f.Name), "Id")
				if assocName == toCamelCase(f.Name) {
					assocName = toCamelCase(toSingular(refParts[0]))
				}
				fields = append(fields, prismaFieldData{
					Name:       assocName,
					PrismaType: relType,
					IsRequired: false,
					Attrs:      []string{relAttr},
				})
				continue
			}
			fields = append(fields, prismaFieldData{
				Name:       toCamelCase(f.Name),
				PrismaType: pt + func() string { if !isReq { return "?" }; return "" }(),
				IsRequired: isReq,
				Attrs:      allAttrs,
			})
		}

		var m2mRels []prismaM2MRelation
		for _, other := range m.ManyToMany {
			jt := joinTableName(m.Name, other)
			otherStruct := toPascalCase(toSingular(other))
			m2mRels = append(m2mRels, prismaM2MRelation{
				Field:     toCamelCase(other),
				ModelName: otherStruct,
				JoinTable: jt,
			})
		}

		var backRel []prismaBackRelation
		for _, br := range backRefs[m.Name] {
			fromStruct := toPascalCase(toSingular(br.fromModel))
			backRel = append(backRel, prismaBackRelation{
				Field:     toCamelCase(br.fromModel),
				ModelName: fromStruct + "[]",
			})
		}

		prismaModels = append(prismaModels, prismaModelData{
			StructName:    structName,
			TableName:     m.Name,
			HasTimestamps: modelHasTimestamps(m),
			Fields:        fields,
			M2MRelations:  m2mRels,
			BackRelations: backRel,
		})
	}

	data := prismaSchemaData{DBProvider: dbProvider, Models: prismaModels}
	var buf strings.Builder
	if err := template.Must(template.New("node_prisma_schema").Parse(nodePrismaSchmaTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute node_prisma_schema template: %w", err)
	}
	return buf.String(), nil
}

// ── dev.sh (node variant) ─────────────────────────────────────────────────────

func GenerateNodeDevScript(cfg *Config) (string, error) {
	data := struct {
		Port    int
		IsMySQL bool
	}{Port: cfg.App.Port, IsMySQL: cfg.Database.Driver == "mysql"}
	var buf strings.Builder
	if err := template.Must(template.New("node_dev_sh").Parse(nodeDevShTmpl)).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute node_dev_sh template: %w", err)
	}
	return buf.String(), nil
}
