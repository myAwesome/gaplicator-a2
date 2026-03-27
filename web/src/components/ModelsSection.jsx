import { useState } from 'react'
import { ChevronRight, Plus } from './icons.jsx'
import ModelCard from './ModelCard.jsx'
import { uid, DEFAULT_MODEL } from '../App.jsx'

export default function ModelsSection({ models, onChange, modelNames }) {
  const [open, setOpen] = useState(true)

  function addModel() {
    onChange(ms => [...ms, DEFAULT_MODEL()])
  }

  function updateModel(id, patch) {
    onChange(ms => ms.map(m => m._id === id ? { ...m, ...patch } : m))
  }

  function deleteModel(id) {
    onChange(ms => ms.filter(m => m._id !== id))
  }

  return (
    <div className="section-card">
      <div
        className={`section-header ${open ? 'open' : ''}`}
        onClick={() => setOpen(o => !o)}
      >
        <ChevronRight className={`section-chevron ${open ? 'open' : ''}`} />
        <span className="section-icon">📦</span>
        <span className="section-title">Models</span>
        <span className="section-badge">{models.length} model{models.length !== 1 ? 's' : ''}</span>
      </div>
      {open && (
        <div className="section-body">
          {models.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state-icon">📦</div>
              No models yet. Add your first model to get started.
            </div>
          ) : (
            <div className="models-list">
              {models.map(model => (
                <ModelCard
                  key={model._id}
                  model={model}
                  onChange={patch => updateModel(model._id, patch)}
                  onDelete={() => deleteModel(model._id)}
                  allModelNames={modelNames}
                />
              ))}
            </div>
          )}
          <button className="btn btn-primary add-model-btn" onClick={addModel}>
            <Plus /> Add Model
          </button>
        </div>
      )}
    </div>
  )
}
