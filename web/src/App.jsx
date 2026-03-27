import { useState, useCallback, useId } from 'react'
import { generateYaml, generateYamlHighlighted } from './yamlGenerator.js'
import AppSection from './components/AppSection.jsx'
import DatabaseSection from './components/DatabaseSection.jsx'
import AuthSection from './components/AuthSection.jsx'
import ModelsSection from './components/ModelsSection.jsx'
import YamlPreview from './components/YamlPreview.jsx'

let _id = 0
export function uid() { return ++_id }

const DEFAULT_FIELD = () => ({
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

const DEFAULT_MODEL = () => ({
  _id: uid(),
  name: '',
  many_to_many: [],
  fields: [DEFAULT_FIELD()],
})

const INITIAL = {
  app: { name: 'my-app', port: 8080 },
  database: { host: 'localhost', port: 5432, name: 'my_db', user: 'postgres', password: 'secret' },
  auth: { enabled: false, model: '' },
  models: [],
}

export { DEFAULT_FIELD, DEFAULT_MODEL }

export default function App() {
  const [config, setConfig] = useState(INITIAL)

  const setApp = useCallback(patch =>
    setConfig(c => ({ ...c, app: { ...c.app, ...patch } })), [])

  const setDatabase = useCallback(patch =>
    setConfig(c => ({ ...c, database: { ...c.database, ...patch } })), [])

  const setAuth = useCallback(patch =>
    setConfig(c => ({ ...c, auth: { ...c.auth, ...patch } })), [])

  const setModels = useCallback(updater =>
    setConfig(c => ({ ...c, models: typeof updater === 'function' ? updater(c.models) : updater })), [])

  const yaml = generateYaml(config)
  const highlighted = generateYamlHighlighted(config)
  const modelNames = config.models.map(m => m.name).filter(Boolean)

  return (
    <div className="app-layout">
      <header className="app-header">
        <div className="app-header-logo">
          Gaplic<span>ator</span>
        </div>
        <div className="app-header-sep" />
        <div className="app-header-sub">Schema Generator</div>
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

        <YamlPreview yaml={yaml} highlighted={highlighted} />
      </div>
    </div>
  )
}
