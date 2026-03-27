import { useState, useCallback } from 'react'
import { Copy, Check } from './icons.jsx'

export default function YamlPreview({ yaml, highlighted }) {
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(yaml)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Fallback for non-https
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
        <span className="preview-title">YAML Preview</span>
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
