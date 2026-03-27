import { useState } from 'react'
import { ChevronRight, Trash, Plus } from './icons.jsx'

const FIELD_TYPES = [
  'varchar(255)', 'varchar(100)', 'varchar(200)', 'text',
  'int', 'bigint', 'smallint', 'float', 'double', 'decimal(10,2)',
  'boolean', 'date', 'datetime', 'timestamp', 'uuid',
  'enum', 'char(2)',
]

const FK_TYPES = ['int', 'bigint', 'uuid']

export default function FieldEditor({ field, onChange, onDelete, modelNames }) {
  const [open, setOpen] = useState(false)
  const isEnum = field.type === 'enum'
  const isFK = FK_TYPES.includes(field.type) || field.references

  function updateEnum(idx, val) {
    const next = [...(field.values || [])]
    next[idx] = val
    onChange({ values: next })
  }

  function addEnumVal() {
    onChange({ values: [...(field.values || []), ''] })
  }

  function removeEnumVal(idx) {
    const next = (field.values || []).filter((_, i) => i !== idx)
    onChange({ values: next })
  }

  // All FK targets: model.id (always valid) + any field from models
  const fkOptions = modelNames.map(m => `${m}.id`)

  return (
    <div className="field-row">
      <div
        className={`field-row-header ${open ? 'open' : ''}`}
        onClick={() => setOpen(o => !o)}
      >
        <ChevronRight className={`section-chevron ${open ? 'open' : ''}`} />
        <span className={`field-row-name ${!field.name ? 'placeholder' : ''}`}>
          {field.name || 'untitled field'}
        </span>
        <span className="field-row-type">{field.type}</span>
        <button
          className="btn-icon danger"
          onClick={e => { e.stopPropagation(); onDelete() }}
          title="Remove field"
        >
          <Trash size={12} />
        </button>
      </div>

      {open && (
        <div className="field-row-body">
          {/* Name + Type */}
          <div className="form-row cols-2">
            <div className="form-group">
              <label className="form-label">Field Name <span className="required">*</span></label>
              <input
                className="form-input"
                value={field.name}
                onChange={e => onChange({ name: e.target.value })}
                placeholder="column_name"
                spellCheck={false}
              />
            </div>
            <div className="form-group">
              <label className="form-label">Type <span className="required">*</span></label>
              <select
                className="form-select"
                value={FIELD_TYPES.includes(field.type) ? field.type : 'custom'}
                onChange={e => {
                  if (e.target.value === 'custom') return
                  onChange({ type: e.target.value, values: e.target.value === 'enum' ? (field.values?.length ? field.values : ['']) : field.values })
                }}
              >
                {FIELD_TYPES.map(t => (
                  <option key={t} value={t}>{t}</option>
                ))}
                {!FIELD_TYPES.includes(field.type) && (
                  <option value="custom">{field.type}</option>
                )}
              </select>
            </div>
          </div>

          {/* Custom type input for varchar(N), decimal(P,S), char(N) */}
          <div className="form-group">
            <label className="form-label">Custom Type Override</label>
            <input
              className="form-input"
              value={field.type}
              onChange={e => onChange({ type: e.target.value })}
              placeholder="e.g. varchar(500), decimal(12,4)"
              spellCheck={false}
            />
          </div>

          {/* Flags */}
          <div className="field-flags">
            {['required', 'unique', 'index'].map(flag => (
              <label key={flag} className="field-flag">
                <input
                  type="checkbox"
                  checked={!!field[flag]}
                  onChange={e => onChange({ [flag]: e.target.checked })}
                />
                {flag}
              </label>
            ))}
          </div>

          {/* Enum values */}
          {isEnum && (
            <div className="form-group">
              <label className="form-label">Enum Values <span className="required">*</span></label>
              <div className="enum-values">
                {(field.values || []).map((v, i) => (
                  <div key={i} className="enum-value-row">
                    <input
                      className="form-input enum-value-input"
                      value={v}
                      onChange={e => updateEnum(i, e.target.value)}
                      placeholder={`value ${i + 1}`}
                    />
                    <button
                      className="btn-icon danger"
                      onClick={() => removeEnumVal(i)}
                      title="Remove"
                    >
                      <Trash size={12} />
                    </button>
                  </div>
                ))}
                <button className="btn btn-ghost btn-sm" onClick={addEnumVal} style={{ alignSelf: 'flex-start' }}>
                  <Plus size={12} /> Add value
                </button>
              </div>
            </div>
          )}

          {/* Default */}
          <div className="form-group">
            <label className="form-label">Default Value</label>
            {isEnum && (field.values || []).filter(Boolean).length > 0 ? (
              <select
                className="form-select"
                value={field.default}
                onChange={e => onChange({ default: e.target.value })}
              >
                <option value="">— none —</option>
                {(field.values || []).filter(Boolean).map(v => (
                  <option key={v} value={v}>{v}</option>
                ))}
              </select>
            ) : (
              <input
                className="form-input"
                value={field.default}
                onChange={e => onChange({ default: e.target.value })}
                placeholder="leave empty for none"
              />
            )}
          </div>

          {/* Foreign key */}
          <div className="form-group">
            <label className="form-label">References (FK)</label>
            {fkOptions.length > 0 ? (
              <select
                className="form-select"
                value={field.references}
                onChange={e => onChange({ references: e.target.value, display_field: e.target.value ? field.display_field : '' })}
              >
                <option value="">— none —</option>
                {fkOptions.map(opt => (
                  <option key={opt} value={opt}>{opt}</option>
                ))}
              </select>
            ) : (
              <input
                className="form-input"
                value={field.references}
                onChange={e => onChange({ references: e.target.value })}
                placeholder="model.field — e.g. users.id"
              />
            )}
          </div>

          {field.references && (
            <div className="form-group">
              <label className="form-label">Display Field</label>
              <input
                className="form-input"
                value={field.display_field}
                onChange={e => onChange({ display_field: e.target.value })}
                placeholder="name"
              />
            </div>
          )}

          {/* Label */}
          <div className="form-group">
            <label className="form-label">UI Label</label>
            <input
              className="form-input"
              value={field.label}
              onChange={e => onChange({ label: e.target.value })}
              placeholder="Human-readable label"
            />
          </div>
        </div>
      )}
    </div>
  )
}
