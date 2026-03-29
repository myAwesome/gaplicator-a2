const PROJECTS_KEY = 'gaplicator_projects'
const OLD_CONFIG_KEY = 'gaplicator_config'

function makeId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 7)
}

function defaultConfig(slug) {
  return {
    app: { name: slug || 'my-app', port: 8080 },
    database: { driver: 'postgres', host: 'localhost', port: 5432, name: (slug || 'my_app').replace(/-/g, '_'), user: 'postgres', password: 'secret' },
    auth: { enabled: false, model: '' },
    models: [],
  }
}

export function loadProjects() {
  try {
    const raw = localStorage.getItem(PROJECTS_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed) && parsed.length > 0) return parsed
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
        config: oldConfig,
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
