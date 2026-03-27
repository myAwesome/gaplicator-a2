import { useState } from 'react'
import { ChevronRight } from './icons.jsx'

export default function DatabaseSection({ db, onChange }) {
  const [open, setOpen] = useState(true)

  return (
    <div className="section-card">
      <div
        className={`section-header ${open ? 'open' : ''}`}
        onClick={() => setOpen(o => !o)}
      >
        <ChevronRight className={`section-chevron ${open ? 'open' : ''}`} />
        <span className="section-icon">🗄️</span>
        <span className="section-title">Database</span>
        <span className="section-badge">required</span>
      </div>
      {open && (
        <div className="section-body">
          <div className="form-row cols-2">
            <div className="form-group">
              <label className="form-label">Host <span className="required">*</span></label>
              <input
                className="form-input"
                value={db.host}
                onChange={e => onChange({ host: e.target.value })}
                placeholder="localhost"
              />
            </div>
            <div className="form-group">
              <label className="form-label">Port</label>
              <input
                className="form-input"
                type="number"
                value={db.port}
                onChange={e => onChange({ port: e.target.value })}
                min={1}
                max={65535}
                placeholder="5432"
              />
            </div>
          </div>
          <div className="form-group">
            <label className="form-label">Database Name <span className="required">*</span></label>
            <input
              className="form-input"
              value={db.name}
              onChange={e => onChange({ name: e.target.value })}
              placeholder="my_db"
            />
          </div>
          <div className="form-row cols-2">
            <div className="form-group">
              <label className="form-label">User</label>
              <input
                className="form-input"
                value={db.user}
                onChange={e => onChange({ user: e.target.value })}
                placeholder="postgres"
              />
            </div>
            <div className="form-group">
              <label className="form-label">Password</label>
              <input
                className="form-input"
                type="password"
                value={db.password}
                onChange={e => onChange({ password: e.target.value })}
                placeholder="secret"
              />
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
