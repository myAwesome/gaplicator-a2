/**
 * Generates a Gaplicator-compatible YAML string from the config state.
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
  lines.push('')

  // ── database ─────────────────────────────────────────────
  lines.push('database:')
  lines.push(`  host: ${yamlStr(config.database.host || 'localhost')}`)
  if (config.database.port && Number(config.database.port) !== 5432) {
    lines.push(`  port: ${Number(config.database.port)}`)
  }
  lines.push(`  name: ${yamlStr(config.database.name || 'my_db')}`)
  if (config.database.user && config.database.user !== 'postgres') {
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
