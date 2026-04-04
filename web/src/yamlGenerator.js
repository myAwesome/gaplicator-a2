/**
 * Generates a gaplicator-compatible YAML string from the config state.
 */

function needsQuotes(val) {
  if (typeof val !== 'string') return false
  // Quote if contains special YAML chars or looks like a number/bool
  return /[:#\[\]{}|>&*!,'"\n]/.test(val) ||
    /^\s|\s$/.test(val) ||
    val === '' ||
    /^(true|false|yes|no|null|~)$/i.test(val) ||
    /^\d/.test(val)
}

function yamlStr(val) {
  if (val === undefined || val === null) return ''
  const s = String(val)
  return needsQuotes(s) ? `"${s.replace(/"/g, '\\"')}"` : s
}

function highlightYaml(raw) {
  // Split into lines and add syntax highlighting spans
  return raw
    .split('\n')
    .map(line => {
      // Comment lines
      if (/^\s*#/.test(line)) {
        return `<span class="yaml-comment">${escHtml(line)}</span>`
      }

      // Key: value lines
      const keyMatch = line.match(/^(\s*)(- )?([a-zA-Z_][a-zA-Z0-9_]*)(\s*:)(\s*)(.*)$/)
      if (keyMatch) {
        const [, indent, dash, key, colon, space, rest] = keyMatch
        const indentHtml = escHtml(indent)
        const dashHtml = dash ? `<span class="yaml-dash">- </span>` : ''
        const keyHtml = `<span class="yaml-key">${escHtml(key)}</span>`
        const colonHtml = escHtml(colon)
        const spaceHtml = escHtml(space)
        const restHtml = colorValue(rest)
        return `${indentHtml}${dashHtml}${keyHtml}${colonHtml}${spaceHtml}${restHtml}`
      }

      // Dash-only list items (e.g. model names under many_to_many)
      const dashOnly = line.match(/^(\s*-\s+)(.*)$/)
      if (dashOnly) {
        const [, prefix, val] = dashOnly
        return `<span class="yaml-dash">${escHtml(prefix)}</span>${colorValue(val)}`
      }

      return escHtml(line)
    })
    .join('\n')
}

function colorValue(v) {
  if (!v) return ''
  const trimmed = v.trim()
  if (trimmed === 'true' || trimmed === 'false') {
    return `<span class="yaml-bool">${escHtml(v)}</span>`
  }
  if (/^\[.*\]$/.test(trimmed)) {
    // Inline array — highlight each value
    return escHtml(v)
  }
  if (/^-?\d+(\.\d+)?$/.test(trimmed)) {
    return `<span class="yaml-num">${escHtml(v)}</span>`
  }
  // String value
  return `<span class="yaml-str">${escHtml(v)}</span>`
}

function escHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

export function generateYaml(config) {
  const lines = []

  // ── app ──────────────────────────────────────────────────
  lines.push('app:')
  lines.push(`  name: ${yamlStr(config.app.name || 'my-app')}`)
  lines.push(`  port: ${Number(config.app.port) || 8080}`)
  if ((config.app.server || 'go') !== 'go') {
    lines.push(`  server: ${yamlStr(config.app.server)}`)
  }
  lines.push('')

  // ── database ─────────────────────────────────────────────
  const driver = config.database.driver || 'postgres'
  const defaultPort = driver === 'mysql' ? 3306 : 5432
  const defaultUser = driver === 'mysql' ? 'root' : 'postgres'
  lines.push('database:')
  if (driver !== 'postgres') {
    lines.push(`  driver: ${yamlStr(driver)}`)
  }
  lines.push(`  host: ${yamlStr(config.database.host || 'localhost')}`)
  if (config.database.port && Number(config.database.port) !== defaultPort) {
    lines.push(`  port: ${Number(config.database.port)}`)
  }
  lines.push(`  name: ${yamlStr(config.database.name || 'my_db')}`)
  if (config.database.user && config.database.user !== defaultUser) {
    lines.push(`  user: ${yamlStr(config.database.user)}`)
  }
  if (config.database.password && config.database.password !== 'secret') {
    lines.push(`  password: ${yamlStr(config.database.password)}`)
  }
  lines.push('')

  // ── auth (optional) ──────────────────────────────────────
  if (config.auth.enabled && config.auth.model) {
    lines.push('auth:')
    lines.push(`  model: ${yamlStr(config.auth.model)}`)
    lines.push('')
  }

  // ── models ───────────────────────────────────────────────
  if (config.models.length > 0) {
    lines.push('models:')
    for (const model of config.models) {
      if (!model.name) continue
      lines.push(`  - name: ${yamlStr(model.name)}`)
      if (model.timestamps) {
        lines.push('    timestamps: true')
      }

      const m2m = model.many_to_many.filter(Boolean)
      if (m2m.length > 0) {
        lines.push(`    many_to_many: [${m2m.join(', ')}]`)
      }

      const validFields = model.fields.filter(f => f.name && f.type)
      if (validFields.length > 0) {
        lines.push('    fields:')
        for (const field of validFields) {
          lines.push(`      - name: ${yamlStr(field.name)}`)
          lines.push(`        type: ${field.type}`)

          if (field.required) lines.push('        required: true')
          if (field.unique) lines.push('        unique: true')
          if (field.index && !field.unique) lines.push('        index: true')

          if (field.default !== '' && field.default !== undefined && field.default !== null) {
            const def = field.default
            const defStr = (def === 'true' || def === 'false' || /^\d+$/.test(String(def)))
              ? def
              : yamlStr(def)
            lines.push(`        default: ${defStr}`)
          }

          if (field.references) {
            lines.push(`        references: ${yamlStr(field.references)}`)
            if (field.display_field) {
              lines.push(`        display_field: ${yamlStr(field.display_field)}`)
            }
          }

          if (field.label) {
            lines.push(`        label: ${yamlStr(field.label)}`)
          }

          if (field.type === 'enum' && field.values && field.values.filter(Boolean).length > 0) {
            lines.push(`        values: [${field.values.filter(Boolean).join(', ')}]`)
          }
        }
      }
      lines.push('')
    }
  }

  // Remove trailing blank lines but keep one
  const raw = lines.join('\n').replace(/\n{3,}/g, '\n\n').trimEnd() + '\n'
  return raw
}

export function generateYamlHighlighted(config) {
  return highlightYaml(generateYaml(config))
}

// ── Simple schema generation ─────────────────────────────────────────────────

function toSimpleType(field) {
  // FK reference → rel
  if (field.references) return 'rel'
  const t = (field.type || '').toLowerCase()
  // enum with values → enum("a","b","c")
  if (t === 'enum' && field.values && field.values.filter(Boolean).length > 0) {
    const vals = field.values.filter(Boolean).map(v => `"${v}"`).join(',')
    return `enum(${vals})`
  }
  // varchar(N) → string
  if (t.startsWith('varchar')) return 'string'
  return t || 'string'
}

function toSimpleFieldName(field) {
  // Strip _id suffix from FK fields for cleaner simple schema output
  if (field.references && field.name.endsWith('_id')) {
    return field.name.slice(0, -3)
  }
  return field.name
}

export function generateSimpleYaml(config) {
  const models = config.models.filter(m => m.name)
  if (models.length === 0) return '# No models defined\n'

  const lines = []
  lines.push('models:')

  for (const model of models) {
    lines.push(`    ${model.name}:`)
    const fields = model.fields.filter(f => f.name && f.type)
    for (const field of fields) {
      const fname = toSimpleFieldName(field)
      const ftype = toSimpleType(field)
      lines.push(`        ${fname}: ${ftype}`)
    }
  }

  // M2M section — deduplicate by sorting pair
  const seen = new Set()
  const m2mEntries = []

  for (const model of models) {
    const m2m = (model.many_to_many || []).filter(Boolean)
    for (const other of m2m) {
      const key = [model.name, other].sort().join('|')
      if (seen.has(key)) continue
      seen.add(key)
      m2mEntries.push({ name: `${model.name}_has_${other}`, a: model.name, b: other })
    }
  }

  if (m2mEntries.length > 0) {
    lines.push('m2m:')
    for (const entry of m2mEntries) {
      lines.push(`    ${entry.name}:`)
      lines.push(`        ${entry.a}: rel`)
      lines.push(`        ${entry.b}: rel`)
    }
  }

  return lines.join('\n') + '\n'
}

export function generateSimpleYamlHighlighted(config) {
  return highlightYaml(generateSimpleYaml(config))
}
