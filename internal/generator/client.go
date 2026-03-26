package generator

import (
	_ "embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates/react_package_json.tmpl
var reactPackageJSONTmpl string

//go:embed templates/react_index_html.tmpl
var reactIndexHTMLTmpl string

//go:embed templates/react_vite_config.tmpl
var reactViteConfigTmpl string

//go:embed templates/react_tsconfig_json.tmpl
var reactTsConfigTmpl string

//go:embed templates/react_main_tsx.tmpl
var reactMainTmpl string

//go:embed templates/react_app_tsx.tmpl
var reactAppTmpl string

//go:embed templates/react_types_ts.tmpl
var reactTypesTmpl string

//go:embed templates/react_api_ts.tmpl
var reactAPITmpl string

//go:embed templates/react_page_tsx.tmpl
var reactPageTmpl string

//go:embed templates/react_app_css.tmpl
var reactAppCSSTmpl string

// ModelStructName returns the PascalCase singular struct name for a model (e.g. "students" → "Student").
func ModelStructName(m Model) string {
	return toPascalCase(toSingular(m.Name))
}

// ModelFileBasename returns the singular snake_case file basename for a model (e.g. "students" → "student").
func ModelFileBasename(m Model) string {
	return toSingular(m.Name)
}

// isDateType returns true if the SQL type is a date-only type (needs YYYY-MM-DD ↔ RFC3339 conversion).
func isDateType(sqlType string) bool {
	return strings.ToLower(sqlType) == "date"
}

// isDatetimeType returns true if the SQL type is datetime or timestamp (uses datetime-local input).
func isDatetimeType(sqlType string) bool {
	lower := strings.ToLower(sqlType)
	return lower == "datetime" || lower == "timestamp"
}

// isEnumType returns true if the SQL type is enum.
func isEnumType(sqlType string) bool {
	return strings.ToLower(sqlType) == "enum"
}

// fieldLabel returns the display label for a field: Label if set, else Name.
func fieldLabel(f Field) string {
	if f.Label != "" {
		return f.Label
	}
	return f.Name
}

// findLabelField returns the best display field name for a model (prefers "name"/"title", else first field).
func findLabelField(m Model) string {
	for _, f := range m.Fields {
		if f.Name == "name" || f.Name == "title" {
			return f.Name
		}
	}
	if len(m.Fields) > 0 {
		return m.Fields[0].Name
	}
	return "id"
}

// sqlTypeToTS maps a SQL type to the corresponding TypeScript type.
func sqlTypeToTS(sqlType string) string {
	lower := strings.ToLower(sqlType)
	switch {
	case strings.HasPrefix(lower, "varchar"), strings.HasPrefix(lower, "char"), lower == "text", lower == "uuid", lower == "enum":
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

// tsInputDefault returns the default form value for a SQL type.
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

// tsInputType returns the HTML input type for a SQL type.
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
	case lower == "datetime", lower == "timestamp":
		return "datetime-local"
	default:
		return "text"
	}
}

func execTmpl(name, src string, data any) string {
	var buf strings.Builder
	template.Must(template.New(name).Parse(src)).Execute(&buf, data) //nolint:errcheck
	return buf.String()
}

// GenerateReactPackageJSON returns package.json for the React client.
func GenerateReactPackageJSON(cfg *Config) string {
	return execTmpl("react_package_json", reactPackageJSONTmpl, struct{ AppName string }{cfg.App.Name})
}

// GenerateReactIndexHTML returns index.html for the React client.
func GenerateReactIndexHTML(cfg *Config) string {
	return execTmpl("react_index_html", reactIndexHTMLTmpl, struct{ AppName string }{cfg.App.Name})
}

// GenerateReactViteConfig returns vite.config.ts for the React client.
func GenerateReactViteConfig(cfg *Config) string {
	type proxyModel struct{ Name string }
	models := make([]proxyModel, len(cfg.Models))
	for i, m := range cfg.Models {
		models[i] = proxyModel{m.Name}
	}
	return execTmpl("react_vite_config", reactViteConfigTmpl, struct {
		Port   int
		Models []proxyModel
	}{cfg.App.Port, models})
}

// GenerateReactTsConfig returns tsconfig.json for the React client.
func GenerateReactTsConfig() string {
	return execTmpl("react_tsconfig_json", reactTsConfigTmpl, nil)
}

// GenerateReactMain returns src/main.tsx for the React client.
func GenerateReactMain() string {
	return execTmpl("react_main_tsx", reactMainTmpl, nil)
}

// GenerateReactAppCSS returns src/app.css with application styles.
func GenerateReactAppCSS() string {
	return execTmpl("react_app_css", reactAppCSSTmpl, nil)
}

// GenerateReactApp returns src/App.tsx with navigation and routes for all models.
func GenerateReactApp(models []Model) string {
	type appModel struct {
		Name       string
		StructName string
	}
	am := make([]appModel, len(models))
	for i, m := range models {
		am[i] = appModel{m.Name, toPascalCase(toSingular(m.Name))}
	}
	return execTmpl("react_app_tsx", reactAppTmpl, struct{ Models []appModel }{am})
}

// GenerateReactTypes returns src/types/{model}.ts with TypeScript interfaces for a model.
func GenerateReactTypes(m Model, allModels []Model) string {
	type typeField struct {
		Name        string
		TSType      string
		Required    bool
		IsOptionalFK bool
	}
	type m2mImport struct {
		RefType    string
		ImportPath string
	}
	type m2mField struct {
		FieldName string
		TSType    string
	}
	type m2mIDField struct {
		IDsField string
	}

	fields := make([]typeField, len(m.Fields))
	for i, f := range m.Fields {
		fields[i] = typeField{f.Name, sqlTypeToTS(f.Type), f.Required, f.References != "" && !f.Required}
	}

	_ = allModels // used for future display_field resolution; M2M types inferred from names
	var m2mImports []m2mImport
	var m2mFields []m2mField
	var m2mIDFields []m2mIDField
	seen := map[string]bool{}
	for _, otherName := range m.ManyToMany {
		otherStruct := toPascalCase(toSingular(otherName))
		if !seen[otherName] {
			seen[otherName] = true
			m2mImports = append(m2mImports, m2mImport{
				RefType:    otherStruct,
				ImportPath: "./" + toSingular(otherName),
			})
		}
		m2mFields = append(m2mFields, m2mField{
			FieldName: otherName,
			TSType:    otherStruct,
		})
		m2mIDFields = append(m2mIDFields, m2mIDField{
			IDsField: toSingular(otherName) + "_ids",
		})
	}

	return execTmpl("react_types_ts", reactTypesTmpl, struct {
		StructName  string
		Fields      []typeField
		M2MImports  []m2mImport
		M2MFields   []m2mField
		M2MIDFields []m2mIDField
	}{
		StructName:  toPascalCase(toSingular(m.Name)),
		Fields:      fields,
		M2MImports:  m2mImports,
		M2MFields:   m2mFields,
		M2MIDFields: m2mIDFields,
	})
}

// GenerateReactAPI returns src/api/{model}.ts with fetch wrappers for a model.
func GenerateReactAPI(m Model) string {
	singular := toSingular(m.Name)
	return execTmpl("react_api_ts", reactAPITmpl, struct {
		StructName string
		Singular   string
		PluralRaw  string
		PluralName string
	}{
		StructName: toPascalCase(singular),
		Singular:   singular,
		PluralRaw:  m.Name,
		PluralName: toPascalCase(m.Name),
	})
}

// --- react page template data types ---

type pageFKImport struct {
	RefStruct   string
	RefSingular string
	RefPlural   string
	OptionsVar  string
	SetterVar   string
}

type pageM2MRelation struct {
	IDsField    string
	AssocField  string
	Label       string
	OptionsVar  string
	SetterVar   string
	RefStruct   string
	RefSingular string
	RefPlural   string
	LabelField  string
}

type pageFormField struct {
	Name    string
	Default string
}

type pageOpenEditField struct {
	Name string
	Expr string
}

type pagePayloadField struct {
	Name string
	Expr string
}

type pageFormInput struct {
	FieldName  string
	Label      string
	IsFK       bool
	IsEnum     bool
	IsCheckbox bool
	IsNumber   bool
	InputType  string
	Required   bool
	OptionsVar string
	LabelField string
	EnumValues []string
}

type pageTableCell struct {
	Expr string
}

type pageTableColumn struct {
	Label string
	Key   string
}

type pageFilterField struct {
	FieldName  string
	Label      string
	IsEnum     bool
	EnumValues []string
	IsFK       bool
	OptionsVar string
	LabelField string
	IsBool     bool
}

type reactPageData struct {
	StructName     string
	Singular       string
	PluralName     string
	ComponentName  string
	ModelName      string
	HasFKs         bool
	HasRefs        bool
	FKImports      []pageFKImport
	FormFields     []pageFormField
	OpenEditFields []pageOpenEditField
	NeedsPayload   bool
	PayloadFields  []pagePayloadField
	FormInputs     []pageFormInput
	TableColumns   []pageTableColumn
	TableCells     []pageTableCell
	HasSearch      bool
	HasFilters     bool
	FilterFields   []pageFilterField
	HasM2M         bool
	M2MRelations   []pageM2MRelation
}

// GenerateReactPage returns src/pages/{Model}Page.tsx with a CRUD table and form for a model.
// allModels is used to resolve FK references for dropdown rendering.
func GenerateReactPage(m Model, allModels []Model) string {
	singular := toSingular(m.Name)
	structName := toPascalCase(singular)
	pluralName := toPascalCase(m.Name)

	// ── FK resolution (deduplicated by referenced model name) ──────────────
	type fkMeta struct {
		field      Field
		refModel   Model
		labelField string
		optionsVar string
		setterVar  string
		refStruct  string
		refSingular string
		refPlural  string
	}
	seenRef := map[string]bool{}
	var fkFields []fkMeta
	for _, f := range m.Fields {
		if f.References == "" {
			continue
		}
		parts := strings.SplitN(f.References, ".", 2)
		refModelName := parts[0]
		if seenRef[refModelName] {
			continue
		}
		for _, rm := range allModels {
			if rm.Name == refModelName {
				seenRef[refModelName] = true
				rs := toSingular(rm.Name)
				rsn := toPascalCase(rs)
				ov := strings.ToLower(rsn[:1]) + rsn[1:] + "Options"
				sv := "set" + rsn + "Options"
				fkFields = append(fkFields, fkMeta{
					field:       f,
					refModel:    rm,
					labelField:  findLabelField(rm),
					optionsVar:  ov,
					setterVar:   sv,
					refStruct:   rsn,
					refSingular: rs,
					refPlural:   toPascalCase(rm.Name),
				})
				break
			}
		}
	}
	// map field name → fkMeta for form rendering
	fkByField := map[string]fkMeta{}
	for _, fk := range fkFields {
		for _, f := range m.Fields {
			if f.References != "" {
				p := strings.SplitN(f.References, ".", 2)
				if p[0] == fk.refModel.Name {
					fkByField[f.Name] = fk
				}
			}
		}
	}

	// ── FKImports ──────────────────────────────────────────────────────────
	fkImports := make([]pageFKImport, len(fkFields))
	for i, fk := range fkFields {
		fkImports[i] = pageFKImport{
			RefStruct:   fk.refStruct,
			RefSingular: fk.refSingular,
			RefPlural:   fk.refPlural,
			OptionsVar:  fk.optionsVar,
			SetterVar:   fk.setterVar,
		}
	}

	// ── FormFields (EMPTY_FORM) ────────────────────────────────────────────
	formFields := make([]pageFormField, len(m.Fields))
	for i, f := range m.Fields {
		defaultVal := tsInputDefault(f.Type)
		if f.References != "" && !f.Required {
			defaultVal = "null"
		}
		formFields[i] = pageFormField{f.Name, defaultVal}
	}

	// ── OpenEditFields ────────────────────────────────────────────────────
	openEditFields := make([]pageOpenEditField, len(m.Fields))
	for i, f := range m.Fields {
		var expr string
		switch {
		case isDateType(f.Type):
			expr = fmt.Sprintf("item.%s ? (item.%s as string).slice(0, 10) : ''", f.Name, f.Name)
		case isDatetimeType(f.Type):
			expr = fmt.Sprintf("item.%s ? (item.%s as string).slice(0, 16) : ''", f.Name, f.Name)
		case f.References != "" && !f.Required:
			expr = fmt.Sprintf("item.%s ?? null", f.Name)
		case f.Required:
			expr = "item." + f.Name
		default:
			expr = fmt.Sprintf("item.%s ?? %s", f.Name, tsInputDefault(f.Type))
		}
		openEditFields[i] = pageOpenEditField{f.Name, expr}
	}

	// ── PayloadFields ─────────────────────────────────────────────────────
	var payloadFields []pagePayloadField
	for _, f := range m.Fields {
		var expr string
		if isDateType(f.Type) {
			expr = fmt.Sprintf("form.%s ? form.%s + 'T00:00:00Z' : form.%s", f.Name, f.Name, f.Name)
		} else if isDatetimeType(f.Type) {
			expr = fmt.Sprintf("form.%s ? form.%s + ':00Z' : form.%s", f.Name, f.Name, f.Name)
		} else {
			continue
		}
		payloadFields = append(payloadFields, pagePayloadField{f.Name, expr})
	}
	needsPayload := len(payloadFields) > 0

	// ── FormInputs ────────────────────────────────────────────────────────
	formInputs := make([]pageFormInput, len(m.Fields))
	for i, f := range m.Fields {
		fi := pageFormInput{FieldName: f.Name, Label: fieldLabel(f), Required: f.Required}
		if fk, isFk := fkByField[f.Name]; isFk {
			fi.IsFK = true
			fi.OptionsVar = fk.optionsVar
			labelF := f.DisplayField
			if labelF == "" {
				labelF = fk.labelField
			}
			fi.LabelField = labelF
		} else if isEnumType(f.Type) {
			fi.IsEnum = true
			fi.EnumValues = f.Values
		} else {
			it := tsInputType(f.Type)
			fi.InputType = it
			fi.IsCheckbox = it == "checkbox"
			fi.IsNumber = it == "number"
		}
		formInputs[i] = fi
	}

	// ── TableColumns / TableCells ─────────────────────────────────────────
	columns := make([]pageTableColumn, len(m.Fields))
	cells := make([]pageTableCell, len(m.Fields))
	for i, f := range m.Fields {
		columns[i] = pageTableColumn{Label: fieldLabel(f), Key: f.Name}
		if fk, isFk := fkByField[f.Name]; isFk {
			labelF := f.DisplayField
			if labelF == "" {
				labelF = fk.labelField
			}
			cells[i] = pageTableCell{fmt.Sprintf("{%s.find(o => o.id === item.%s)?.%s ?? String(item.%s)}", fk.optionsVar, f.Name, labelF, f.Name)}
		} else if isEnumType(f.Type) {
			cells[i] = pageTableCell{fmt.Sprintf(`<span className={"badge badge--" + (item.%s as string)}>{item.%s}</span>`, f.Name, f.Name)}
		} else if sqlTypeToTS(f.Type) == "boolean" {
			cells[i] = pageTableCell{fmt.Sprintf("{item.%s ? 'yes' : 'no'}", f.Name)}
		} else if isDateType(f.Type) {
			cells[i] = pageTableCell{fmt.Sprintf("{item.%s ? (item.%s as string).slice(0, 10) : ''}", f.Name, f.Name)}
		} else if isDatetimeType(f.Type) {
			cells[i] = pageTableCell{fmt.Sprintf("{item.%s ? (item.%s as string).slice(0, 16).replace('T', ' ') : ''}", f.Name, f.Name)}
		} else {
			cells[i] = pageTableCell{"{item." + f.Name + "}"}
		}
	}

	// ── FilterFields (search + enum, FK, bool dropdowns) ─────────────────────
	var filterFields []pageFilterField
	hasSearch := false
	for _, f := range m.Fields {
		lower := strings.ToLower(f.Type)
		isText := strings.HasPrefix(lower, "text") || strings.HasPrefix(lower, "varchar") || strings.HasPrefix(lower, "char")
		if isText {
			hasSearch = true
			continue
		}
		if fk, isFk := fkByField[f.Name]; isFk {
			labelF := f.DisplayField
			if labelF == "" {
				labelF = fk.labelField
			}
			filterFields = append(filterFields, pageFilterField{
				FieldName:  f.Name,
				Label:      fieldLabel(f),
				IsFK:       true,
				OptionsVar: fk.optionsVar,
				LabelField: labelF,
			})
		} else if isEnumType(f.Type) {
			filterFields = append(filterFields, pageFilterField{
				FieldName:  f.Name,
				Label:      fieldLabel(f),
				IsEnum:     true,
				EnumValues: f.Values,
			})
		} else if lower == "boolean" || lower == "bool" {
			filterFields = append(filterFields, pageFilterField{
				FieldName: f.Name,
				Label:     fieldLabel(f),
				IsBool:    true,
			})
		}
	}
	hasFilters := hasSearch || len(filterFields) > 0

	// ── M2M relations ─────────────────────────────────────────────────────
	var m2mRelations []pageM2MRelation
	seenM2M := map[string]bool{}
	for _, otherName := range m.ManyToMany {
		if seenM2M[otherName] {
			continue
		}
		seenM2M[otherName] = true
		otherSingular := toSingular(otherName)
		otherStruct := toPascalCase(otherSingular)
		ov := strings.ToLower(otherStruct[:1]) + otherStruct[1:] + "Options"
		sv := "set" + otherStruct + "Options"
		labelField := "id"
		for _, rm := range allModels {
			if rm.Name == otherName {
				labelField = findLabelField(rm)
				break
			}
		}
		m2mRelations = append(m2mRelations, pageM2MRelation{
			IDsField:    otherSingular + "_ids",
			AssocField:  otherName,
			Label:       otherName,
			OptionsVar:  ov,
			SetterVar:   sv,
			RefStruct:   otherStruct,
			RefSingular: otherSingular,
			RefPlural:   toPascalCase(otherName),
			LabelField:  labelField,
		})
	}

	hasRefs := len(fkFields) > 0 || len(m2mRelations) > 0

	data := reactPageData{
		StructName:     structName,
		Singular:       singular,
		PluralName:     pluralName,
		ComponentName:  structName + "Page",
		ModelName:      m.Name,
		HasFKs:         len(fkFields) > 0,
		HasRefs:        hasRefs,
		FKImports:      fkImports,
		FormFields:     formFields,
		OpenEditFields: openEditFields,
		NeedsPayload:   needsPayload,
		PayloadFields:  payloadFields,
		FormInputs:     formInputs,
		TableColumns:   columns,
		TableCells:     cells,
		HasSearch:      hasSearch,
		HasFilters:     hasFilters,
		FilterFields:   filterFields,
		HasM2M:         len(m2mRelations) > 0,
		M2MRelations:   m2mRelations,
	}

	return execTmpl("react_page_tsx", reactPageTmpl, data)
}
