package generator

import (
	"fmt"
	"strings"
)

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
	default:
		return "text"
	}
}

// GenerateReactPackageJSON returns package.json for the React client.
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

// GenerateReactIndexHTML returns index.html for the React client.
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

// GenerateReactViteConfig returns vite.config.ts for the React client.
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

// GenerateReactTsConfig returns tsconfig.json for the React client.
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

// GenerateReactMain returns src/main.tsx for the React client.
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

// GenerateReactApp returns src/App.tsx with navigation and routes for all models.
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

// GenerateReactTypes returns src/types/{model}.ts with TypeScript interfaces for a model.
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

// GenerateReactAPI returns src/api/{model}.ts with fetch wrappers for a model.
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

// GenerateReactPage returns src/pages/{Model}Page.tsx with a CRUD table and form for a model.
func GenerateReactPage(m Model) string {
	singular := toSingular(m.Name)
	structName := toPascalCase(singular)
	pluralName := toPascalCase(m.Name)
	componentName := structName + "Page"

	var sb strings.Builder

	// Imports
	fmt.Fprintf(&sb, "import { useState, useEffect } from 'react';\n")
	fmt.Fprintf(&sb, "import type { %s, Create%sInput } from '../types/%s';\n", structName, structName, singular)
	fmt.Fprintf(&sb, "import { list%s, create%s, update%s, delete%s } from '../api/%s';\n\n", pluralName, structName, structName, structName, singular)

	// EMPTY_FORM
	fmt.Fprintf(&sb, "const EMPTY_FORM: Create%sInput = {\n", structName)
	for _, f := range m.Fields {
		fmt.Fprintf(&sb, "  %s: %s,\n", f.Name, tsInputDefault(f.Type))
	}
	sb.WriteString("};\n\n")

	// Component
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
		if isDateType(f.Type) {
			fmt.Fprintf(&sb, "      %s: item.%s ? (item.%s as string).slice(0, 10) : '',\n", f.Name, f.Name, f.Name)
		} else if f.Required {
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
	// Collect date fields that need YYYY-MM-DD → RFC3339 conversion
	var dateFields []Field
	for _, f := range m.Fields {
		if isDateType(f.Type) {
			dateFields = append(dateFields, f)
		}
	}
	if len(dateFields) > 0 {
		sb.WriteString("      const payload = {\n")
		sb.WriteString("        ...form,\n")
		for _, f := range dateFields {
			fmt.Fprintf(&sb, "        %s: form.%s ? form.%s + 'T00:00:00Z' : form.%s,\n", f.Name, f.Name, f.Name, f.Name)
		}
		sb.WriteString("      };\n")
		fmt.Fprintf(&sb, "      if (editing) await update%s(editing.id, payload);\n", structName)
		fmt.Fprintf(&sb, "      else await create%s(payload);\n", structName)
	} else {
		fmt.Fprintf(&sb, "      if (editing) await update%s(editing.id, form);\n", structName)
		fmt.Fprintf(&sb, "      else await create%s(form);\n", structName)
	}
	sb.WriteString("      setShowForm(false); load();\n")
	sb.WriteString("    } catch (e) { console.error(e); }\n")
	sb.WriteString("  }\n\n")

	sb.WriteString("  async function handleDelete(id: number) {\n")
	sb.WriteString("    if (!confirm('Delete?')) return;\n")
	fmt.Fprintf(&sb, "    try { await delete%s(id); load(); } catch (e) { console.error(e); }\n", structName)
	sb.WriteString("  }\n\n")

	// JSX
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
