import { useState, useCallback, useEffect } from 'react'
import {
  generateYaml, generateYamlHighlighted,
  generateSimpleYaml, generateSimpleYamlHighlighted,
} from './yamlGenerator.js'
import AppSection from './components/AppSection.jsx'
import DatabaseSection from './components/DatabaseSection.jsx'
import AuthSection from './components/AuthSection.jsx'
import ModelsSection from './components/ModelsSection.jsx'
import YamlPreview from './components/YamlPreview.jsx'
import ThemeToggle from './components/ThemeToggle.jsx'
import ProjectsView from './components/ProjectsView.jsx'
import { loadProjects, saveProjects, createProject } from './projectStorage.js'

let _id = 0
export function uid() { return ++_id }

export const DEFAULT_FIELD = () => ({
  _id: uid(),
  name: '',
  type: 'varchar(255)',
  required: false,
  unique: false,
  default: '',
  index: false,
  references: '',
  display_field: '',
  label: '',
  values: [],
})

export const DEFAULT_MODEL = () => ({
  _id: uid(),
  name: '',
  many_to_many: [],
  fields: [DEFAULT_FIELD()],
})

export default function App() {
  const [projects, setProjects] = useState(loadProjects)
  const [activeProjectId, setActiveProjectId] = useState(null)

  // Persist projects whenever they change
  useEffect(() => {
    saveProjects(projects)
  }, [projects])

  const activeProject = projects.find(p => p.id === activeProjectId) ?? null

  // ── Project management ──────────────────────────────────────────────────────

  const handleCreate = useCallback((name, type) => {
    const project = createProject(name, type)
    setProjects(ps => [...ps, project])
    setActiveProjectId(project.id)
  }, [])

  const handleOpen = useCallback((id) => {
    setActiveProjectId(id)
  }, [])

  const handleDelete = useCallback((id) => {
    setProjects(ps => ps.filter(p => p.id !== id))
    if (activeProjectId === id) setActiveProjectId(null)
  }, [activeProjectId])

  const handleBack = useCallback(() => {
    setActiveProjectId(null)
  }, [])

  // ── Config updater for active project ───────────────────────────────────────

  const updateConfig = useCallback((updater) => {
    setProjects(ps => ps.map(p => {
      if (p.id !== activeProjectId) return p
      const newConfig = typeof updater === 'function' ? updater(p.config) : updater
      return { ...p, config: newConfig, updatedAt: new Date().toISOString() }
    }))
  }, [activeProjectId])

  const setApp = useCallback(patch =>
    updateConfig(c => ({ ...c, app: { ...c.app, ...patch } })), [updateConfig])

  const setDatabase = useCallback(patch =>
    updateConfig(c => ({ ...c, database: { ...c.database, ...patch } })), [updateConfig])

  const setAuth = useCallback(patch =>
    updateConfig(c => ({ ...c, auth: { ...c.auth, ...patch } })), [updateConfig])

  const setModels = useCallback(updater =>
    updateConfig(c => ({ ...c, models: typeof updater === 'function' ? updater(c.models) : updater })),
    [updateConfig])

  // ── Projects view ───────────────────────────────────────────────────────────

  if (!activeProject) {
    return (
      <ProjectsView
        projects={projects}
        onOpen={handleOpen}
        onCreate={handleCreate}
        onDelete={handleDelete}
      />
    )
  }

  // ── Editor view ─────────────────────────────────────────────────────────────

  const config = activeProject.config
  const schemaType = activeProject.type

  const fullYaml = generateYaml(config)
  const fullHighlighted = generateYamlHighlighted(config)
  const simpleYaml = generateSimpleYaml(config)
  const simpleHighlighted = generateSimpleYamlHighlighted(config)

  const modelNames = config.models.map(m => m.name).filter(Boolean)

  return (
    <div className="app-layout">
      <header className="app-header">
        <button className="btn-back" onClick={handleBack} title="Back to projects">
          ←
        </button>
        <div className="app-header-logo">
          Gaplic<span>ator</span>
        </div>
        <div className="app-header-sep" />
        <div className="app-header-project">
          <span className="app-header-project-name">{activeProject.name}</span>
          <span className={`project-type-badge ${schemaType}`}>
            {schemaType === 'simple' ? 'Simple' : 'Full'}
          </span>
        </div>
        <ThemeToggle />
      </header>

      <div className="app-body">
        <div className="form-panel">
          <AppSection app={config.app} onChange={setApp} />
          <DatabaseSection db={config.database} onChange={setDatabase} />
          <AuthSection auth={config.auth} modelNames={modelNames} onChange={setAuth} />
          <ModelsSection
            models={config.models}
            onChange={setModels}
            modelNames={modelNames}
          />
        </div>

        <YamlPreview
          fullYaml={fullYaml}
          fullHighlighted={fullHighlighted}
          simpleYaml={simpleYaml}
          simpleHighlighted={simpleHighlighted}
          defaultTab={schemaType}
        />
      </div>
    </div>
  )
}
