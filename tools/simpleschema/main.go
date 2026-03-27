// simpleschema converts a compact YAML schema into the full gaplicator config format.
//
// Simple schema format:
//
//	models:
//	  student:
//	    name:       string
//	    city:       enum("Roma","Milan","London")
//	    rank:       float
//	    dob:        date
//	    active:     boolean
//	  discipline:
//	    name:       string
//	  lesson:
//	    name:       string
//	    date:       date
//	    discipline: rel
//	m2m:
//	  student_has_lessons:
//	    student: rel
//	    lesson:  rel
//
// Supported field types:
//   - string              → varchar(255)
//   - boolean             → boolean
//   - float               → float
//   - date                → date
//   - datetime            → datetime
//   - int                 → int
//   - text                → text
//   - enum("a","b","c")   → enum with values list
//   - rel                 → bigint FK to the model named by the field key
//
// m2m entries: each junction table with two rel fields generates a
// many_to_many declaration on the first model.
//
// Usage:
//
//	simpleschema [flags] input.yaml [output.yaml]
//
// Flags:
//
//	-app    application name (default "my-app")
//	-port   application port (default 8080)
//	-dbhost database host    (default "localhost")
//	-dbname database name    (default "<app>_db")
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Output types — must match the fields accepted by gaplicator's working schema
// ---------------------------------------------------------------------------

type outConfig struct {
	App      outApp     `yaml:"app"`
	Database outDB      `yaml:"database"`
	Models   []outModel `yaml:"models"`
}

type outApp struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}

type outDB struct {
	Host string `yaml:"host"`
	Name string `yaml:"name"`
}

type outModel struct {
	Name       string     `yaml:"name"`
	Fields     []outField `yaml:"fields"`
	ManyToMany []string   `yaml:"many_to_many,omitempty"`
}

type outField struct {
	Name         string   `yaml:"name"`
	Type         string   `yaml:"type"`
	Required     bool     `yaml:"required,omitempty"`
	Unique       bool     `yaml:"unique,omitempty"`
	Values       []string `yaml:"values,omitempty"`
	References   string   `yaml:"references,omitempty"`
	DisplayField string   `yaml:"display_field,omitempty"`
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	appName := flag.String("app", "my-app", "application name")
	appPort := flag.Int("port", 8080, "application port")
	dbHost := flag.String("dbhost", "localhost", "database host")
	dbName := flag.String("dbname", "", "database name (default: <app>_db)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: simpleschema [flags] input.yaml [output.yaml]")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		fatalf("read %s: %v", args[0], err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		fatalf("parse yaml: %v", err)
	}

	if *dbName == "" {
		*dbName = strings.ReplaceAll(*appName, "-", "_") + "_db"
	}

	cfg, err := convert(&doc, *appName, *appPort, *dbHost, *dbName)
	if err != nil {
		fatalf("convert: %v", err)
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		fatalf("marshal: %v", err)
	}

	if len(args) >= 2 {
		if err := os.WriteFile(args[1], out, 0o644); err != nil {
			fatalf("write %s: %v", args[1], err)
		}
		fmt.Fprintf(os.Stderr, "written to %s\n", args[1])
	} else {
		os.Stdout.Write(out)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "simpleschema: "+format+"\n", args...)
	os.Exit(1)
}

// ---------------------------------------------------------------------------
// Conversion
// ---------------------------------------------------------------------------

type simpleField struct{ name, typ string }
type simpleModel struct {
	name   string
	fields []simpleField
}

func convert(doc *yaml.Node, appName string, port int, dbHost, dbName string) (*outConfig, error) {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("unexpected yaml document structure")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("yaml root must be a mapping")
	}

	var modelsNode, m2mNode *yaml.Node
	for i := 0; i+1 < len(root.Content); i += 2 {
		switch root.Content[i].Value {
		case "models":
			modelsNode = root.Content[i+1]
		case "m2m":
			m2mNode = root.Content[i+1]
		}
	}

	if modelsNode == nil {
		return nil, fmt.Errorf("missing 'models' section")
	}
	if modelsNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("'models' must be a mapping")
	}

	rawModels, err := parseModels(modelsNode)
	if err != nil {
		return nil, err
	}

	// Build set of known model names (singular) for reference validation.
	modelSet := make(map[string]bool, len(rawModels))
	for _, m := range rawModels {
		modelSet[m.name] = true
	}

	// Build display-field index: singular model name → first string-like field name.
	// Used when generating FK display_field.
	displayField := buildDisplayIndex(rawModels)

	// m2mByModel[singularModel] = []pluralOtherModel
	m2mByModel, err := parseM2M(m2mNode, modelSet)
	if err != nil {
		return nil, err
	}

	// Convert to working models.
	outModels := make([]outModel, 0, len(rawModels))
	for _, sm := range rawModels {
		fields, err := convertFields(sm.fields, modelSet, displayField)
		if err != nil {
			return nil, fmt.Errorf("model %q: %w", sm.name, err)
		}
		outModels = append(outModels, outModel{
			Name:       pluralize(sm.name),
			Fields:     fields,
			ManyToMany: m2mByModel[sm.name],
		})
	}

	return &outConfig{
		App:      outApp{Name: appName, Port: port},
		Database: outDB{Host: dbHost, Name: dbName},
		Models:   outModels,
	}, nil
}

func parseModels(node *yaml.Node) ([]simpleModel, error) {
	var models []simpleModel
	for i := 0; i+1 < len(node.Content); i += 2 {
		modelName := node.Content[i].Value
		fieldsNode := node.Content[i+1]
		if fieldsNode.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("model %q: expected a mapping of fields", modelName)
		}
		fields := make([]simpleField, 0, len(fieldsNode.Content)/2)
		for j := 0; j+1 < len(fieldsNode.Content); j += 2 {
			fields = append(fields, simpleField{
				name: fieldsNode.Content[j].Value,
				typ:  strings.TrimSpace(fieldsNode.Content[j+1].Value),
			})
		}
		models = append(models, simpleModel{name: modelName, fields: fields})
	}
	return models, nil
}

// buildDisplayIndex returns the "best" display field for each model:
// prefers "name", then "title", then the first non-rel field.
func buildDisplayIndex(models []simpleModel) map[string]string {
	idx := make(map[string]string, len(models))
	for _, m := range models {
		best := "name" // safe default
		for _, f := range m.fields {
			if f.name == "name" || f.name == "title" {
				best = f.name
				break
			}
		}
		// If no name/title, pick first non-rel field.
		if best == "name" {
			found := false
			for _, f := range m.fields {
				if f.name == "name" {
					found = true
					break
				}
			}
			if !found {
				for _, f := range m.fields {
					if strings.TrimSpace(f.typ) != "rel" {
						best = f.name
						break
					}
				}
			}
		}
		idx[m.name] = best
	}
	return idx
}

// parseM2M returns a map from singular model name → []plural other model names.
// The first rel field in each junction entry is considered the "owner" side.
func parseM2M(node *yaml.Node, modelSet map[string]bool) (map[string][]string, error) {
	byModel := make(map[string][]string)
	if node == nil {
		return byModel, nil
	}
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("'m2m' must be a mapping")
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		junctionName := node.Content[i].Value
		relsNode := node.Content[i+1]
		if relsNode.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("m2m entry %q: expected a mapping", junctionName)
		}
		var rels []string
		for j := 0; j+1 < len(relsNode.Content); j += 2 {
			if strings.TrimSpace(relsNode.Content[j+1].Value) == "rel" {
				rels = append(rels, relsNode.Content[j].Value)
			}
		}
		if len(rels) < 2 {
			return nil, fmt.Errorf("m2m entry %q: need at least 2 rel fields, got %d", junctionName, len(rels))
		}
		m1, m2 := rels[0], rels[1]
		if !modelSet[m1] {
			return nil, fmt.Errorf("m2m entry %q: unknown model %q", junctionName, m1)
		}
		if !modelSet[m2] {
			return nil, fmt.Errorf("m2m entry %q: unknown model %q", junctionName, m2)
		}
		byModel[m1] = append(byModel[m1], pluralize(m2))
	}
	return byModel, nil
}

func convertFields(fields []simpleField, modelSet map[string]bool, displayIdx map[string]string) ([]outField, error) {
	out := make([]outField, 0, len(fields))
	for _, f := range fields {
		switch {
		case f.typ == "rel":
			if !modelSet[f.name] {
				return nil, fmt.Errorf("field %q: rel references unknown model %q", f.name, f.name)
			}
			df, ok := displayIdx[f.name]
			if !ok {
				df = "name"
			}
			out = append(out, outField{
				Name:         f.name + "_id",
				Type:         "bigint",
				References:   pluralize(f.name) + ".id",
				DisplayField: df,
			})

		case strings.HasPrefix(f.typ, "enum("):
			vals := parseEnum(f.typ)
			if len(vals) == 0 {
				return nil, fmt.Errorf("field %q: invalid enum syntax %q", f.name, f.typ)
			}
			out = append(out, outField{
				Name:   f.name,
				Type:   "enum",
				Values: vals,
			})

		case f.typ == "string":
			out = append(out, outField{Name: f.name, Type: "varchar(255)"})

		case f.typ == "boolean", f.typ == "bool",
			f.typ == "float", f.typ == "double",
			f.typ == "int", f.typ == "bigint", f.typ == "smallint",
			f.typ == "date", f.typ == "datetime", f.typ == "timestamp",
			f.typ == "text", f.typ == "uuid":
			out = append(out, outField{Name: f.name, Type: f.typ})

		default:
			// Pass through unknown types (e.g. varchar(100), decimal(10,2)).
			out = append(out, outField{Name: f.name, Type: f.typ})
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var enumRe = regexp.MustCompile(`^enum\((.+)\)$`)

func parseEnum(s string) []string {
	m := enumRe.FindStringSubmatch(s)
	if len(m) < 2 {
		return nil
	}
	var vals []string
	for _, part := range strings.Split(m[1], ",") {
		v := strings.Trim(strings.TrimSpace(part), `"'`)
		if v != "" {
			vals = append(vals, v)
		}
	}
	return vals
}

// pluralize applies simple English pluralization rules.
func pluralize(word string) string {
	switch {
	case strings.HasSuffix(word, "ch"),
		strings.HasSuffix(word, "sh"),
		strings.HasSuffix(word, "ss"):
		return word + "es"
	case strings.HasSuffix(word, "x"),
		strings.HasSuffix(word, "z"):
		return word + "es"
	case strings.HasSuffix(word, "y") && len(word) > 1 && !isVowel(word[len(word)-2]):
		return word[:len(word)-1] + "ies"
	case strings.HasSuffix(word, "f") && !strings.HasSuffix(word, "ff"):
		return word[:len(word)-1] + "ves"
	case strings.HasSuffix(word, "fe"):
		return word[:len(word)-2] + "ves"
	default:
		return word + "s"
	}
}

func isVowel(b byte) bool {
	return b == 'a' || b == 'e' || b == 'i' || b == 'o' || b == 'u'
}
