import { useState, useCallback } from 'react'
import { Copy, Check } from './icons.jsx'

export default function YamlPreview({ fullYaml, fullHighlighted, simpleYaml, simpleHighlighted, defaultTab }) {
  const [tab, setTab] = useState(defaultTab || 'full')
  const [copied, setCopied] = useState(false)

  const yaml = tab === 'simple' ? simpleYaml : fullYaml
  const highlighted = tab === 'simple' ? simpleHighlighted : fullHighlighted

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(yaml)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      const el = document.createElement('textarea')
      el.value = yaml
      el.style.position = 'fixed'
      el.style.opacity = '0'
      document.body.appendChild(el)
      el.select()
      document.execCommand('copy')
      document.body.removeChild(el)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }, [yaml])

  return (
    <div className="preview-panel">
      <div className="preview-header">
        <div className="preview-tabs">
          <button
            className={`preview-tab ${tab === 'simple' ? 'active' : ''}`}
            onClick={() => setTab('simple')}
          >
            Simple
          </button>
          <button
            className={`preview-tab ${tab === 'full' ? 'active' : ''}`}
            onClick={() => setTab('full')}
          >
            Full
          </button>
        </div>
        <button
          className={`btn-copy ${copied ? 'copied' : ''}`}
          onClick={handleCopy}
          title="Copy to clipboard"
        >
          {copied ? <Check size={13} /> : <Copy size={13} />}
          {copied ? 'Copied!' : 'Copy'}
        </button>
      </div>
      <div
        className="preview-code"
        dangerouslySetInnerHTML={{ __html: highlighted }}
      />
    </div>
  )
}
