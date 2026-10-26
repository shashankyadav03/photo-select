import { useState } from 'react'

export function CopyCommand({ command, label = 'Copy' }) {
  const [copied, setCopied] = useState(false)

  async function copy(e) {
    const box = e.currentTarget.parentElement.querySelector('code')
    try {
      await navigator.clipboard.writeText(command)
    } catch {
      // Older browsers: select the text so Ctrl/⌘+C works.
      const range = document.createRange()
      range.selectNodeContents(box)
      const sel = window.getSelection()
      sel.removeAllRanges()
      sel.addRange(range)
      document.execCommand?.('copy')
    }
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="command">
      <code>{command}</code>
      <button type="button" className={copied ? 'copy copied' : 'copy'} onClick={copy}>
        {copied ? '✓ Copied' : label}
      </button>
    </div>
  )
}

export function Tabs({ tabs, value, onChange }) {
  return (
    <div className="tabs" role="tablist">
      {Object.entries(tabs).map(([key, name]) => (
        <button
          key={key}
          type="button"
          role="tab"
          aria-selected={value === key}
          className={value === key ? 'tab active' : 'tab'}
          onClick={() => onChange(key)}
        >
          {name}
        </button>
      ))}
    </div>
  )
}

export function Steps({ children }) {
  return <ol className="steps">{children}</ol>
}

export function Note({ kind = 'info', children }) {
  return <div className={`note note-${kind}`}>{children}</div>
}

export function Kbd({ children }) {
  return <kbd>{children}</kbd>
}

export function formatSize(bytes) {
  if (!bytes) return ''
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

export function DownloadButton({ manifest, label, children, primary }) {
  const file = manifest?.files?.[label]
  if (!file) {
    return (
      <span className="button disabled" title="Downloads appear once the site is deployed">
        {children}
      </span>
    )
  }
  return (
    <a className={primary ? 'button primary' : 'button'} href={`/downloads/${file.name}`} download>
      ⬇ {children} <small>{formatSize(file.size)}</small>
    </a>
  )
}
