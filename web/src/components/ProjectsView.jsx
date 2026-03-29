import { useState } from 'react'
import ThemeToggle from './ThemeToggle.jsx'
import { relativeTime } from '../projectStorage.js'

function NewProjectForm({ onSubmit, onCancel }) {
  const [name, setName] = useState('')
  const [type, setType] = useState('full')

  function handleSubmit(e) {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) return
    onSubmit(trimmed, type)
  }

  return (
    <div className="new-project-modal-overlay" onClick={onCancel}>
      <div className="new-project-modal" onClick={e => e.stopPropagation()}>
        <div className="new-project-modal-title">New Project</div>
        <form onSubmit={handleSubmit}>
          <div className="form-group" style={{ marginBottom: 14 }}>
            <label className="form-label">Project Name</label>
            <input
              className="form-input"
              placeholder="my-app"
              value={name}
              onChange={e => setName(e.target.value)}
              autoFocus
            />
          </div>

          <div className="form-group" style={{ marginBottom: 20 }}>
            <label className="form-label">Schema Type</label>
            <div className="schema-type-picker">
              <button
                type="button"
                className={`schema-type-btn ${type === 'simple' ? 'active' : ''}`}
                onClick={() => setType('simple')}
              >
                <span className="schema-type-icon">⚡</span>
                <span className="schema-type-label">Simple</span>
                <span className="schema-type-desc">Compact YAML, auto-inferred</span>
              </button>
              <button
                type="button"
                className={`schema-type-btn ${type === 'full' ? 'active' : ''}`}
                onClick={() => setType('full')}
              >
                <span className="schema-type-icon">⚙</span>
                <span className="schema-type-label">Full</span>
                <span className="schema-type-desc">Complete control over fields</span>
              </button>
            </div>
          </div>

          <div className="new-project-modal-actions">
            <button type="button" className="btn btn-ghost" onClick={onCancel}>Cancel</button>
            <button type="submit" className="btn btn-primary" disabled={!name.trim()}>
              Create Project
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function ProjectCard({ project, onOpen, onDelete }) {
  const modelCount = project.config?.models?.length ?? 0

  function handleDelete(e) {
    e.stopPropagation()
    if (confirm(`Delete "${project.name}"? This cannot be undone.`)) {
      onDelete(project.id)
    }
  }

  return (
    <div className="project-card" onClick={() => onOpen(project.id)}>
      <div className="project-card-header">
        <span className="project-card-name">{project.name}</span>
        <span className={`project-type-badge ${project.type}`}>
          {project.type === 'simple' ? 'Simple' : 'Full'}
        </span>
      </div>
      <div className="project-card-meta">
        <span>{modelCount} {modelCount === 1 ? 'model' : 'models'}</span>
        <span className="project-card-dot">·</span>
        <span>{relativeTime(project.updatedAt)}</span>
      </div>
      <div className="project-card-footer">
        <button
          className="btn btn-primary btn-sm"
          onClick={e => { e.stopPropagation(); onOpen(project.id) }}
        >
          Open
        </button>
        <button
          className="btn btn-danger btn-sm"
          onClick={handleDelete}
        >
          Delete
        </button>
      </div>
    </div>
  )
}

export default function ProjectsView({ projects, onOpen, onCreate, onDelete }) {
  const [showForm, setShowForm] = useState(false)

  function handleCreate(name, type) {
    onCreate(name, type)
    setShowForm(false)
  }

  return (
    <div className="app-layout">
      <header className="app-header">
        <div className="app-header-logo">
          Gaplic<span>ator</span>
        </div>
        <div className="app-header-sep" />
        <div className="app-header-sub">Schema Generator</div>
        <ThemeToggle />
      </header>

      <div className="projects-page">
        <div className="projects-page-header">
          <h1 className="projects-page-title">Projects</h1>
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>
            + New Project
          </button>
        </div>

        {projects.length === 0 ? (
          <div className="empty-state" style={{ marginTop: 60 }}>
            <div className="empty-state-icon">📂</div>
            <div>No projects yet. Create one to get started.</div>
          </div>
        ) : (
          <div className="projects-grid">
            {projects.map(p => (
              <ProjectCard
                key={p.id}
                project={p}
                onOpen={onOpen}
                onDelete={onDelete}
              />
            ))}
          </div>
        )}
      </div>

      {showForm && (
        <NewProjectForm
          onSubmit={handleCreate}
          onCancel={() => setShowForm(false)}
        />
      )}
    </div>
  )
}
