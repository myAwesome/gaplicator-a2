const PROJECTS_KEY = 'gaplicator_projects'
const OLD_CONFIG_KEY = 'gaplicator_config'

function makeId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 7)
}

function defaultConfig(slug) {
  return {
    app: { name: slug || 'my-app', port: 8080, server: 'go' },
    database: { driver: 'postgres', host: 'localhost', port: 5432, name: (slug || 'my_app').replace(/-/g, '_'), user: 'postgres', password: 'secret' },
    auth: { enabled: false, model: '' },
    models: [],
  }
}

function normalizeConfig(config = {}) {
  const driver = config.database?.driver || 'postgres'
  const defaultPort = driver === 'mysql' ? 3306 : 5432
  const defaultUser = driver === 'mysql' ? 'root' : 'postgres'

  const app = {
    name: config.app?.name ?? 'my-app',
    port: config.app?.port ?? 8080,
    server: config.app?.server || 'go',
  }

  const database = {
    driver,
    host: config.database?.host ?? 'localhost',
    port: config.database?.port ?? defaultPort,
    name: config.database?.name ?? 'my_db',
    user: config.database?.user ?? defaultUser,
    password: config.database?.password ?? 'secret',
  }

  const auth = {
    enabled: !!config.auth?.enabled,
    model: config.auth?.model || '',
  }

  const models = Array.isArray(config.models)
    ? config.models.map(model => ({
      ...model,
      timestamps: !!model.timestamps,
      many_to_many: Array.isArray(model.many_to_many) ? model.many_to_many : [],
      fields: Array.isArray(model.fields) ? model.fields : [],
    }))
    : []

  return { app, database, auth, models }
}

function normalizeProject(project) {
  return {
    ...project,
    config: normalizeConfig(project.config),
  }
}

export function loadProjects() {
  try {
    const raw = localStorage.getItem(PROJECTS_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed) && parsed.length > 0) return parsed.map(normalizeProject)
    }
  } catch (_) {}

  // Migrate from old single-config storage
  try {
    const oldRaw = localStorage.getItem(OLD_CONFIG_KEY)
    if (oldRaw) {
      const oldConfig = JSON.parse(oldRaw)
      const migrated = [{
        id: makeId(),
        name: oldConfig.app?.name || 'My Project',
        type: 'full',
        config: normalizeConfig(oldConfig),
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      }]
      saveProjects(migrated)
      return migrated
    }
  } catch (_) {}

  return []
}

export function saveProjects(projects) {
  try {
    localStorage.setItem(PROJECTS_KEY, JSON.stringify(projects))
  } catch (_) {}
}

export function createProject(name, type) {
  const slug = name.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-_]/g, '') || 'my-app'
  return {
    id: makeId(),
    name,
    type, // 'simple' | 'full'
    config: defaultConfig(slug),
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  }
}

export function relativeTime(isoStr) {
  const diff = Date.now() - new Date(isoStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days === 1) return 'yesterday'
  if (days < 30) return `${days}d ago`
  return new Date(isoStr).toLocaleDateString()
}
