import React, { useEffect, useRef, useState } from 'react'
import { addTorrent } from '../services/api'

function syntaxHighlightJson(rawJsonText) {
  const escaped = rawJsonText
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  return escaped.replace(
    /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(?=:))|("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*")|\b(true|false)\b|\bnull\b|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?/g,
    (match, keyToken, _, stringToken, __, boolToken) => {
      if (keyToken) return `<span class="json-key">${match}</span>`
      if (stringToken) return `<span class="json-string">${match}</span>`
      if (boolToken) return `<span class="json-boolean">${match}</span>`
      if (match === 'null') return `<span class="json-null">${match}</span>`
      return `<span class="json-number">${match}</span>`
    }
  )
}

export default function FetchApiActions() {
  const jellyfinUrl = import.meta.env.VITE_JELLYFIN_URL || 'https://watch.k8s.iaminyourpc.xyz'
  const jellyfinAccent = '#6b5fda'
  const [movieName, setMovieName] = useState('')
  const [saveLocation, setSaveLocation] = useState('')
  const [qbittorrentCategory, setQbittorrentCategory] = useState('')
  const [qbittorrentTags, setQbittorrentTags] = useState('')
  const [findSubs, setFindSubs] = useState(false)
  const [manage, setManage] = useState(true)
  const [urlError, setUrlError] = useState('')
  const [jsonText, setJsonText] = useState('{}')
  const [buttonState, setButtonState] = useState('idle') // idle | pending | success
  const timersRef = useRef([])
  const typingTimersRef = useRef([])

  useEffect(() => {
    return () => {
      timersRef.current.forEach(clearTimeout)
      timersRef.current = []
      typingTimersRef.current.forEach(clearTimeout)
      typingTimersRef.current = []
    }
  }, [])

  const isValidUrl = (value) => {
    try {
      const parsed = new URL(value)
      return parsed.protocol === 'http:' || parsed.protocol === 'https:'
    } catch {
      return false
    }
  }

  const queueTimeout = (fn, delay) => {
    const id = setTimeout(fn, delay)
    timersRef.current.push(id)
  }

  const clearTypingTimers = () => {
    typingTimersRef.current.forEach(clearTimeout)
    typingTimersRef.current = []
  }

  const animateJsonOutput = (data) => {
    clearTypingTimers()
    const fullText = JSON.stringify(data ?? {}, null, 2)
    const lines = fullText.split('\n')
    setJsonText(lines[0] || '{}')

    lines.slice(1).forEach((line, index) => {
      const id = setTimeout(() => {
        setJsonText((prev) => `${prev}\n${line}`)
      }, (index + 1) * 62)
      typingTimersRef.current.push(id)
    })
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    const value = movieName.trim()

    if (!isValidUrl(value)) {
      setUrlError('Invalid URL')
      clearTypingTimers()
      setJsonText('{}')
      return
    }

    setUrlError('')
    setButtonState('pending')

    try {
      const tags = qbittorrentTags
        .split(',')
        .map((tag) => tag.trim())
        .filter(Boolean)

      const payload = {
        url: value,
        save_path: saveLocation.trim(),
        category: qbittorrentCategory.trim(),
        tags,
        manage,
        find_subs: findSubs,
      }

      const response = await addTorrent(payload)
      animateJsonOutput(response)
    } catch (error) {
      clearTypingTimers()
      animateJsonOutput({ error: error.message || 'Request failed' })
      setButtonState('idle')
      return
    }
    setButtonState('success')
    queueTimeout(() => {
      setButtonState('idle')
    }, 1000)
  }

  return (
    <div>
      <div className="relative max-w-2xl bg-white rounded-xl shadow-[0_6px_14px_rgba(15,23,42,0.12)] p-6 border border-gray-100">
        <a
          href={jellyfinUrl}
          target="_blank"
          rel="noreferrer"
          className="absolute top-6 right-6"
          aria-label="Open Jellyfin"
        >
          <img
            src="https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/jellyfin.png"
            alt="Jellyfin"
            className="w-8 h-8"
          />
        </a>
        <h3 className="text-xl font-semibold text-gray-900 mb-2">Import to Jellyfin</h3>
        <p className="text-gray-600 text-sm mb-6">Your movie/show will be waiting for you.</p>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <input
              type="text"
              value={movieName}
              onChange={(e) => {
                setMovieName(e.target.value)
                if (urlError) {
                  setUrlError('')
                }
              }}
              className={`w-full px-4 py-2 border rounded-lg outline-none transition-colors ${
                urlError
                  ? 'border-red-500 focus:ring-2 focus:ring-red-500 focus:border-red-500'
                  : 'border-gray-300 focus:ring-2 focus:ring-[#6b5fda] focus:border-transparent'
              }`}
              placeholder="Torrent URL"
            />
            {urlError && <p className="mt-2 text-sm text-red-600">Invalid URL</p>}
          </div>

          <div>
            <input
              type="text"
              value={saveLocation}
              onChange={(e) => setSaveLocation(e.target.value)}
              className="w-full px-4 py-2 border rounded-lg outline-none transition-colors border-gray-300 focus:ring-2 focus:ring-[#6b5fda] focus:border-transparent"
              placeholder="Save Location"
            />
          </div>

          <div>
            <input
              type="text"
              value={qbittorrentCategory}
              onChange={(e) => setQbittorrentCategory(e.target.value)}
              className="w-full px-4 py-2 border rounded-lg outline-none transition-colors border-gray-300 focus:ring-2 focus:ring-[#6b5fda] focus:border-transparent"
              placeholder="qBittorrent Category"
            />
          </div>

          <div>
            <input
              type="text"
              value={qbittorrentTags}
              onChange={(e) => setQbittorrentTags(e.target.value)}
              className="w-full px-4 py-2 border rounded-lg outline-none transition-colors border-gray-300 focus:ring-2 focus:ring-[#6b5fda] focus:border-transparent"
              placeholder="Comma-separated qBittorrent Tags"
            />
          </div>

          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-5">
              <div className="inline-flex items-center select-none gap-3">
                <button
                  type="button"
                  role="switch"
                  aria-checked={manage}
                  aria-label="Manage"
                  onClick={() => setManage((prev) => !prev)}
                  className={`relative h-7 w-12 shrink-0 overflow-hidden rounded-full border transition-colors duration-300 focus:outline-none focus:ring-2 focus:ring-[#6b5fda] focus:ring-offset-1 ${
                    manage ? '' : 'bg-slate-300 border-slate-300'
                  }`}
                  style={manage ? { backgroundColor: jellyfinAccent, borderColor: jellyfinAccent } : undefined}
                >
                  <span
                    className={`absolute left-[2px] top-0.5 h-[23px] w-[23px] rounded-full bg-white shadow-md transition-transform duration-300 ${
                      manage ? 'translate-x-[21px]' : 'translate-x-0'
                    }`}
                  />
                </button>
                <span className={`text-sm font-medium transition-colors duration-200 ${manage ? 'text-[#6b5fda]' : 'text-gray-800'}`}>
                  Manage
                </span>
              </div>

              <div className="inline-flex items-center select-none gap-3">
                <button
                  type="button"
                  role="switch"
                  aria-checked={findSubs}
                  aria-label="Find subs"
                  onClick={() => setFindSubs((prev) => !prev)}
                  className={`relative h-7 w-12 shrink-0 overflow-hidden rounded-full border transition-colors duration-300 focus:outline-none focus:ring-2 focus:ring-[#6b5fda] focus:ring-offset-1 ${
                    findSubs ? '' : 'bg-slate-300 border-slate-300'
                  }`}
                  style={findSubs ? { backgroundColor: jellyfinAccent, borderColor: jellyfinAccent } : undefined}
                >
                  <span
                    className={`absolute left-[2px] top-0.5 h-[23px] w-[23px] rounded-full bg-white shadow-md transition-transform duration-300 ${
                      findSubs ? 'translate-x-[21px]' : 'translate-x-0'
                    }`}
                  />
                </button>
                <span className={`text-sm font-medium transition-colors duration-200 ${findSubs ? 'text-[#6b5fda]' : 'text-gray-800'}`}>
                  Find subs
                </span>
              </div>
            </div>

            <button
              type="submit"
              disabled={buttonState === 'pending'}
              className={`relative w-[30%] min-w-[170px] overflow-hidden rounded-lg font-medium py-3 px-4 flex items-center justify-center transition-all duration-300 shadow-[inset_0_1px_0_rgba(255,255,255,0.26),inset_0_-2px_6px_rgba(0,0,0,0.2)] ${
                buttonState === 'success'
                  ? 'text-gray-900'
                  : 'text-white'
              }`}
              style={{ backgroundColor: jellyfinAccent }}
            >
              {buttonState === 'success' && (
                <span className="btn-splash-down absolute inset-0 bg-[#4fd68f]" />
              )}
              <span className="relative z-10 block w-full h-6">
                {buttonState === 'success' ? (
                  <span className="absolute inset-0 flex items-center justify-center gap-2">
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                    </svg>
                    <span>Success</span>
                  </span>
                ) : buttonState === 'pending' ? (
                  <span className="absolute inset-0">
                    <span className="absolute left-[16.67%] top-1/2 -translate-y-1/2 -translate-x-1/2">Fire</span>
                    <span className="btn-spinner absolute right-[9px]" aria-hidden="true" />
                  </span>
                ) : (
                  <>
                    <span className="absolute left-[16.67%] top-1/2 -translate-y-1/2 -translate-x-1/2">Fire</span>
                    <svg className="absolute right-[9px] top-1/2 -translate-y-1/2 w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M7 7l5 5-5 5M12 7l5 5-5 5" />
                    </svg>
                  </>
                )}
              </span>
            </button>
          </div>

          <div className="mt-6 w-full rounded-xl bg-black p-4 min-h-[220px] border border-gray-800">
            <pre
              className="text-[12px] leading-5 font-mono whitespace-pre-wrap break-all text-slate-300"
              dangerouslySetInnerHTML={{ __html: syntaxHighlightJson(jsonText) }}
            />
          </div>
        </form>
      </div>
    </div>
  )
}
