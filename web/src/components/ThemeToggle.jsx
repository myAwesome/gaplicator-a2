import { useState, useEffect } from 'react'
import { Sun, Moon } from './icons.jsx'

function getInitialTheme() {
  try {
    const stored = localStorage.getItem('theme')
    if (stored === 'light' || stored === 'dark') return stored
  } catch (_) {}
  return 'dark'
}

export default function ThemeToggle() {
  const [theme, setTheme] = useState(getInitialTheme)

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    try { localStorage.setItem('theme', theme) } catch (_) {}
  }, [theme])

  const isLight = theme === 'light'

  return (
    <button
      className="btn-icon theme-toggle"
      onClick={() => setTheme(isLight ? 'dark' : 'light')}
      title={isLight ? 'Switch to dark mode' : 'Switch to light mode'}
    >
      {isLight ? <Moon /> : <Sun />}
    </button>
  )
}
