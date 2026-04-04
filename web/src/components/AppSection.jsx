import { useState } from 'react'
import { ChevronRight } from './icons.jsx'

export default function AppSection({ app, onChange }) {
  const [open, setOpen] = useState(true)

  return (
    <div className="section-card">
      <div
        className={`section-header ${open ? 'open' : ''}`}
        onClick={() => setOpen(o => !o)}
      >
        <ChevronRight className={`section-chevron ${open ? 'open' : ''}`} />
        <span className="section-icon">⚙️</span>
        <span className="section-title">App</span>
        <span className="section-badge">required</span>
      </div>
      {open && (
        <div className="section-body">
          <div className="form-row cols-2">
            <div className="form-group">
              <label className="form-label">Name <span className="required">*</span></label>
              <input
                className="form-input"
                value={app.name}
                onChange={e => onChange({ name: e.target.value })}
                placeholder="my-app"
                spellCheck={false}
              />
            </div>
            <div className="form-group">
              <label className="form-label">Port <span className="required">*</span></label>
              <input
                className="form-input"
                type="number"
                value={app.port}
                onChange={e => onChange({ port: e.target.value })}
                min={1}
                max={65535}
                placeholder="8080"
              />
            </div>
          </div>
          <div className="form-group">
            <label className="form-label">Server</label>
            <select
              className="form-select"
              value={app.server || 'go'}
              onChange={e => onChange({ server: e.target.value })}
            >
              <option value="go">Go (Gin + GORM)</option>
              <option value="node">Node.js (Express + Prisma)</option>
            </select>
          </div>
        </div>
      )}
    </div>
  )
}
