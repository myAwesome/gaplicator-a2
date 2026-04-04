import { useState } from 'react'
import { ChevronRight, Trash, Plus } from './icons.jsx'
import FieldEditor from './FieldEditor.jsx'
import { uid, DEFAULT_FIELD } from '../App.jsx'

export default function ModelCard({ model, onChange, onDelete, allModelNames }) {
  const [open, setOpen] = useState(true)

  // Other model names (excluding self) for many_to_many selection
  const otherModels = allModelNames.filter(n => n && n !== model.name)

  function toggleM2M(name) {
    const current = model.many_to_many || []
    if (current.includes(name)) {
      onChange({ many_to_many: current.filter(n => n !== name) })
    } else {
      onChange({ many_to_many: [...current, name] })
    }
  }

  function updateField(id, patch) {
    onChange({
      fields: model.fields.map(f => f._id === id ? { ...f, ...patch } : f),
    })
  }

  function deleteField(id) {
    onChange({ fields: model.fields.filter(f => f._id !== id) })
  }

  function addField() {
    onChange({ fields: [...model.fields, DEFAULT_FIELD()] })
  }

  const fieldCount = model.fields.filter(f => f.name).length

  return (
    <div className="model-card">
      <div
        className={`model-card-header ${open ? 'open' : ''}`}
        onClick={() => setOpen(o => !o)}
      >
        <ChevronRight className={`section-chevron ${open ? 'open' : ''}`} />
        <span className={`model-card-name ${!model.name ? 'placeholder' : ''}`}>
          {model.name || 'untitled model'}
        </span>
        <span className="model-card-meta">
          {fieldCount} {fieldCount === 1 ? 'field' : 'fields'}
        </span>
        <button
          className="btn-icon danger"
          onClick={e => { e.stopPropagation(); onDelete() }}
          title="Remove model"
        >
          <Trash size={13} />
        </button>
      </div>

      {open && (
        <div className="model-card-body">
          {/* Model name */}
          <div className="form-group">
            <label className="form-label">Model Name <span className="required">*</span></label>
            <input
              className="form-input"
              value={model.name}
              onChange={e => onChange({ name: e.target.value })}
              placeholder="plural_snake_case — e.g. blog_posts"
              spellCheck={false}
            />
          </div>

          <label
            className="toggle-switch"
            onClick={() => onChange({ timestamps: !model.timestamps })}
          >
            <div className={`toggle-track ${model.timestamps ? 'on' : ''}`}>
              <div className="toggle-thumb" />
            </div>
            <span className="toggle-label">Enable timestamps (`created_at`, `updated_at`, `deleted_at`)</span>
          </label>

          {/* Many-to-many */}
          {otherModels.length > 0 && (
            <div className="form-group">
              <label className="form-label">Many-to-Many</label>
              <div className="m2m-chips">
                {(model.many_to_many || []).filter(Boolean).map(n => (
                  <div key={n} className="m2m-chip">
                    {n}
                    <button onClick={() => toggleM2M(n)} title="Remove">×</button>
                  </div>
                ))}
                {(model.many_to_many || []).length === 0 && (
                  <span className="m2m-empty">none selected</span>
                )}
              </div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 6 }}>
                {otherModels
                  .filter(n => !(model.many_to_many || []).includes(n))
                  .map(n => (
                    <button
                      key={n}
                      className="btn btn-ghost btn-sm"
                      onClick={() => toggleM2M(n)}
                    >
                      + {n}
                    </button>
                  ))}
              </div>
            </div>
          )}

          {/* Fields */}
          <div className="section-divider">Fields</div>

          <div className="fields-list">
            {model.fields.length === 0 ? (
              <div className="empty-state">
                <div className="empty-state-icon">📋</div>
                No fields yet
              </div>
            ) : (
              model.fields.map(field => (
                <FieldEditor
                  key={field._id}
                  field={field}
                  onChange={patch => updateField(field._id, patch)}
                  onDelete={() => deleteField(field._id)}
                  modelNames={allModelNames.filter(n => n !== model.name)}
                />
              ))
            )}
          </div>

          <button className="btn btn-ghost btn-sm" onClick={addField}>
            <Plus size={12} /> Add field
          </button>
        </div>
      )}
    </div>
  )
}
