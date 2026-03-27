import { useState } from 'react'
import { ChevronRight } from './icons.jsx'

export default function AuthSection({ auth, modelNames, onChange }) {
  const [open, setOpen] = useState(false)

  return (
    <div className="section-card">
      <div
        className={`section-header ${open ? 'open' : ''}`}
        onClick={() => setOpen(o => !o)}
      >
        <ChevronRight className={`section-chevron ${open ? 'open' : ''}`} />
        <span className="section-icon">🔐</span>
        <span className="section-title">Auth</span>
        <span className={`section-badge ${auth.enabled ? '' : 'optional'}`}>
          {auth.enabled ? 'JWT enabled' : 'optional'}
        </span>
      </div>
      {open && (
        <div className="section-body">
          <label
            className="toggle-switch"
            onClick={() => onChange({ enabled: !auth.enabled })}
          >
            <div className={`toggle-track ${auth.enabled ? 'on' : ''}`}>
              <div className="toggle-thumb" />
            </div>
            <span className="toggle-label">Enable JWT authentication</span>
          </label>

          {auth.enabled && (
            <div className="form-group">
              <label className="form-label">Auth Model <span className="required">*</span></label>
              {modelNames.length > 0 ? (
                <select
                  className="form-select"
                  value={auth.model}
                  onChange={e => onChange({ model: e.target.value })}
                >
                  <option value="">— select model —</option>
                  {modelNames.map(n => (
                    <option key={n} value={n}>{n}</option>
                  ))}
                  <option value="users">users (auto-created)</option>
                </select>
              ) : (
                <input
                  className="form-input"
                  value={auth.model}
                  onChange={e => onChange({ model: e.target.value })}
                  placeholder="users"
                />
              )}
              <span style={{ fontSize: 11, color: 'var(--text-dim)', marginTop: 3 }}>
                If the model is not in the list below, Gaplicator will auto-create it with default fields.
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
