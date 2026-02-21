import React, { useEffect, useRef, useState } from 'react'
import { addTorrent, getTorrents } from '../services/api'

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
  const ICON_TRANSITION_MS = 240
  const SUCCESS_ICON_HOLD_MS = 2000
  const DEFAULT_SAVE_LOCATION = '/data/Windows/Movies'
  const DEFAULT_CATEGORY = 'jellyfin'
  const DEFAULT_TAGS_PLACEHOLDER = 'fetch-api,another tag'
  const DEFAULT_TAGS_FALLBACK = 'fetch-api'
  const jellyfinUrl = '/__jellyfin__'
  const qbittorrentUrl = '/__qbittorrent__'
  const jellyfinAccent = '#4f6df6'
  const jellyfinAccentRgb = '79, 109, 246'
  const [movieName, setMovieName] = useState('')
  const [saveLocation, setSaveLocation] = useState('')
  const [qbittorrentCategory, setQbittorrentCategory] = useState('')
  const [qbittorrentTags, setQbittorrentTags] = useState('')
  const [showOptions, setShowOptions] = useState(false)
  const [findSubs, setFindSubs] = useState(false)
  const [notify, setNotify] = useState(true)
  const [urlError, setUrlError] = useState('')
  const [jsonText, setJsonText] = useState('{}')
  const [buttonState, setButtonState] = useState('idle') // idle | pending
  const [buttonIcon, setButtonIcon] = useState('arrows') // arrows | pending | check
  const [iconTransition, setIconTransition] = useState(null)
  const [torrents, setTorrents] = useState([])
  const [hasItems, setHasItems] = useState(false)
  const timersRef = useRef([])
  const typingTimersRef = useRef([])
  const iconFlowRef = useRef(0)
  const buttonIconRef = useRef('arrows')
  const isSubmitting = buttonState === 'pending'

  useEffect(() => {
    return () => {
      timersRef.current.forEach(clearTimeout)
      timersRef.current = []
      typingTimersRef.current.forEach(clearTimeout)
      typingTimersRef.current = []
    }
  }, [])

  useEffect(() => {
    const fetchTorrents = async () => {
      try {
        const data = await getTorrents()
        const items = data.items || []
        setTorrents(items)
        setHasItems(items.length > 0)
      } catch (e) {
        // silently ignore polling errors
      }
    }
    fetchTorrents()
    const id = setInterval(fetchTorrents, hasItems ? 5000 : 10000)
    return () => clearInterval(id)
  }, [hasItems])

  const wait = (delay) => {
    return new Promise((resolve) => {
      const id = setTimeout(resolve, delay)
      timersRef.current.push(id)
    })
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

  const transitionButtonIcon = async (nextIcon, flowId) => {
    const currentIcon = buttonIconRef.current
    if (currentIcon === nextIcon) {
      return
    }

    setIconTransition({ from: currentIcon, to: nextIcon })
    await wait(ICON_TRANSITION_MS)

    if (flowId !== iconFlowRef.current) {
      return
    }

    buttonIconRef.current = nextIcon
    setButtonIcon(nextIcon)
    setIconTransition(null)
  }

  const renderButtonIcon = (iconName) => {
    if (iconName === 'check') {
      return (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
        </svg>
      )
    }

    if (iconName === 'pending') {
      return <span className="btn-pending-dot" />
    }

    return (
      <span
        aria-hidden="true"
        className="block w-5 h-5"
        style={{
          backgroundColor: 'currentColor',
          WebkitMaskImage: 'url(https://i.imgur.com/QHbctJc.png)',
          maskImage: 'url(https://i.imgur.com/QHbctJc.png)',
          WebkitMaskSize: 'contain',
          maskSize: 'contain',
          WebkitMaskRepeat: 'no-repeat',
          maskRepeat: 'no-repeat',
          WebkitMaskPosition: 'center',
          maskPosition: 'center',
        }}
      />
    )
  }

  const labelForState = (stateName) => {
    if (stateName === 'pending') return 'Importing'
    if (stateName === 'check') return 'Imported'
    return 'Import'
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (buttonState === 'pending') {
      return
    }
    const value = movieName.trim()

    if (!value) {
      setUrlError('* Torrent URL is required')
      clearTypingTimers()
      setJsonText('{}')
      return
    }

    setUrlError('')
    setButtonState('pending')
    const flowId = iconFlowRef.current + 1
    iconFlowRef.current = flowId
    await transitionButtonIcon('pending', flowId)

    let requestSucceeded = false
    try {
      const effectiveSaveLocation = saveLocation.trim() || DEFAULT_SAVE_LOCATION
      const effectiveCategory = qbittorrentCategory.trim() || DEFAULT_CATEGORY
      const effectiveTags = (qbittorrentTags.trim() || DEFAULT_TAGS_FALLBACK)
        .split(',')
        .map((tag) => tag.trim())
        .filter(Boolean)

      const payload = {
        url: value,
        save_path: effectiveSaveLocation,
        category: effectiveCategory,
        tags: effectiveTags,
        notify,
        find_subs: findSubs,
      }

      const response = await addTorrent(payload)
      animateJsonOutput(response)
      requestSucceeded = true
    } catch (error) {
      clearTypingTimers()
      animateJsonOutput({ error: error.message || 'Request failed' })
    }
    setMovieName('')

    if (!requestSucceeded) {
      await transitionButtonIcon('arrows', flowId)
      if (flowId === iconFlowRef.current) {
        setButtonState('idle')
      }
      return
    }

    await transitionButtonIcon('check', flowId)
    if (flowId === iconFlowRef.current) {
      setButtonState('idle')
    }
    await wait(SUCCESS_ICON_HOLD_MS)
    await transitionButtonIcon('arrows', flowId)
  }

  return (
    <div>
      <div
        className="relative max-w-2xl bg-white rounded-xl shadow-[0_6px_14px_rgba(15,23,42,0.12)] p-6 border border-gray-100"
        style={{ '--card-accent': jellyfinAccent, '--card-accent-rgb': jellyfinAccentRgb }}
      >
        <div className="mb-2 flex items-center justify-between">
          <h3 className="text-xl font-semibold text-gray-900">Add to Jellyfin</h3>
          <div className="flex items-center gap-4">
            <a
              href={jellyfinUrl}
              target="_blank"
              rel="noreferrer"
              aria-label="Open Jellyfin"
              className="inline-flex items-center text-sm font-medium"
              style={{ color: jellyfinAccent }}
            >
              <span
                aria-hidden="true"
                className="block w-4 h-4"
                style={{
                  backgroundColor: jellyfinAccent,
                  WebkitMaskImage: 'url(https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/jellyfin.png)',
                  maskImage: 'url(https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/jellyfin.png)',
                  WebkitMaskSize: 'contain',
                  maskSize: 'contain',
                  WebkitMaskRepeat: 'no-repeat',
                  maskRepeat: 'no-repeat',
                  WebkitMaskPosition: 'center',
                  maskPosition: 'center',
                }}
              />
            </a>
            <a
              href={qbittorrentUrl}
              target="_blank"
              rel="noreferrer"
              aria-label="Open qBittorrent"
              className="inline-flex items-center text-sm font-medium"
              style={{ color: jellyfinAccent }}
            >
              <span
                aria-hidden="true"
                className="block w-4 h-4"
                style={{
                  backgroundColor: jellyfinAccent,
                  WebkitMaskImage: 'url(https://i.imgur.com/3uMVbI9.png)',
                  maskImage: 'url(https://i.imgur.com/3uMVbI9.png)',
                  WebkitMaskSize: 'contain',
                  maskSize: 'contain',
                  WebkitMaskRepeat: 'no-repeat',
                  maskRepeat: 'no-repeat',
                  WebkitMaskPosition: 'center',
                  maskPosition: 'center',
                }}
              />
            </a>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="mt-5 space-y-5">
          <div>
            <input
              type="text"
              value={movieName}
              disabled={isSubmitting}
              onChange={(e) => {
                setMovieName(e.target.value)
                if (urlError) {
                  setUrlError('')
                }
              }}
              className="w-full px-4 py-[7px] text-[14px] border rounded-md outline-none transition-colors border-gray-300 animated-focus-input disabled:cursor-not-allowed disabled:opacity-60"
              style={urlError ? { borderColor: jellyfinAccent } : undefined}
              placeholder="Magnet or url"
            />
            {urlError && <p className="mt-2 text-sm font-semibold" style={{ color: jellyfinAccent }}>{urlError}</p>}
          </div>

          <div className="space-y-3">
            <button
              type="button"
              disabled={isSubmitting}
              onClick={() => setShowOptions((prev) => !prev)}
              className="inline-flex items-center gap-2 text-sm font-semibold disabled:opacity-60 disabled:cursor-not-allowed"
              style={{ color: jellyfinAccent }}
              aria-expanded={showOptions}
              aria-controls="import-options"
            >
              <svg
                className={`w-4 h-4 transition-transform duration-250 ${showOptions ? 'rotate-180' : 'rotate-0'}`}
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M6 9l6 6 6-6" />
              </svg>
              <span>Options</span>
            </button>

            <div id="import-options" className={`options-panel ${showOptions ? 'options-panel-open' : ''}`}>
              <div className="options-panel-inner space-y-5">
                <div>
                  <p className="mb-2 text-sm font-medium text-gray-700">Save Location</p>
                  <input
                    type="text"
                    value={saveLocation}
                    disabled={isSubmitting}
                    onChange={(e) => setSaveLocation(e.target.value)}
                    className="w-full px-4 py-[7px] text-[14px] border rounded-md outline-none transition-colors border-gray-300 animated-focus-input disabled:cursor-not-allowed disabled:opacity-60"
                    placeholder={DEFAULT_SAVE_LOCATION}
                  />
                </div>

                <div>
                  <p className="mb-2 text-sm font-medium text-gray-700">Category</p>
                  <input
                    type="text"
                    value={qbittorrentCategory}
                    disabled={isSubmitting}
                    onChange={(e) => setQbittorrentCategory(e.target.value)}
                    className="w-full px-4 py-[7px] text-[14px] border rounded-md outline-none transition-colors border-gray-300 animated-focus-input disabled:cursor-not-allowed disabled:opacity-60"
                    placeholder={DEFAULT_CATEGORY}
                  />
                </div>

                <div>
                  <p className="mb-2 text-sm font-medium text-gray-700">Tags</p>
                  <input
                    type="text"
                    value={qbittorrentTags}
                    disabled={isSubmitting}
                    onChange={(e) => setQbittorrentTags(e.target.value)}
                    className="w-full px-4 py-[7px] text-[14px] border rounded-md outline-none transition-colors border-gray-300 animated-focus-input disabled:cursor-not-allowed disabled:opacity-60"
                    placeholder={DEFAULT_TAGS_PLACEHOLDER}
                  />
                </div>
              </div>
            </div>
          </div>

          <div className="-mt-3 flex items-start justify-between gap-4">
            <div className="option-list" style={{ '--option-accent': jellyfinAccent }}>
              <button
                type="button"
                role="checkbox"
                disabled={isSubmitting}
                aria-checked={notify}
                aria-label="Notify"
                onClick={() => setNotify((prev) => !prev)}
                className="option-row-btn"
              >
                <span className="option-check-shell">
                  <span className={`option-check ${notify ? 'option-check-on' : ''}`}>
                    <svg viewBox="0 0 16 16" className={`option-check-mark ${notify ? 'option-check-mark-on' : ''}`} aria-hidden="true">
                      <path d="M3.4 8.4 6.6 11.4 12.6 4.9" />
                    </svg>
                  </span>
                </span>
                <span className={`option-copy ${notify ? 'option-copy-with-subtitle' : ''}`}>
                  <span className="option-title">Notify</span>
                  <span className={`option-subtitle ${notify ? 'option-subtitle-visible' : ''}`}>In Slack when completed</span>
                </span>
              </button>

              <div className="pt-4">
                <button
                  type="button"
                  role="checkbox"
                  disabled={isSubmitting}
                  aria-checked={findSubs}
                  aria-label="Subtitles"
                  onClick={() => setFindSubs((prev) => !prev)}
                  className="option-row-btn"
                >
                  <span className="option-check-shell">
                    <span className={`option-check ${findSubs ? 'option-check-on' : ''}`}>
                      <svg viewBox="0 0 16 16" className={`option-check-mark ${findSubs ? 'option-check-mark-on' : ''}`} aria-hidden="true">
                        <path d="M3.4 8.4 6.6 11.4 12.6 4.9" />
                      </svg>
                    </span>
                  </span>
                  <span className={`option-copy ${findSubs ? 'option-copy-with-subtitle' : ''}`}>
                    <span className="option-title">Subtitles</span>
                    <span className={`option-subtitle ${findSubs ? 'option-subtitle-visible' : ''}`}>Find and download when completed</span>
                  </span>
                </button>
              </div>
            </div>

            <button
              type="submit"
              disabled={buttonState === 'pending'}
              className="import-action-btn relative w-[30%] min-w-[130px] overflow-hidden rounded-md font-semibold py-2.5 px-3 flex items-center justify-center transition-all duration-300 text-white disabled:cursor-not-allowed disabled:opacity-70"
              style={{ backgroundColor: jellyfinAccent }}
            >
              <span className="relative z-10 inline-flex items-center justify-center gap-2">
                <span className="btn-label-stack" aria-hidden="true">
                  {iconTransition ? (
                    <>
                      <span className="btn-label-layer btn-icon-exit">{labelForState(iconTransition.from)}</span>
                      <span className="btn-label-layer btn-icon-enter">{labelForState(iconTransition.to)}</span>
                    </>
                  ) : (
                    <span className="btn-label-layer">{labelForState(buttonIcon)}</span>
                  )}
                </span>
                <span className="btn-icon-stack" aria-hidden="true">
                  {iconTransition ? (
                    <>
                      <span className="btn-icon-layer btn-icon-exit">{renderButtonIcon(iconTransition.from)}</span>
                      <span className="btn-icon-layer btn-icon-enter">{renderButtonIcon(iconTransition.to)}</span>
                    </>
                  ) : (
                    <span className="btn-icon-layer">{renderButtonIcon(buttonIcon)}</span>
                  )}
                </span>
              </span>
            </button>
          </div>

          <div className="mt-3 w-full rounded-lg bg-black p-4 min-h-[220px] border border-gray-800">
            <pre
              className="text-[12px] leading-5 whitespace-pre-wrap break-all text-slate-300"
              dangerouslySetInnerHTML={{ __html: syntaxHighlightJson(jsonText) }}
            />
          </div>

          {torrents.length === 0 ? (
            <p className="mt-4 text-[15px] font-semibold" style={{ color: jellyfinAccent }}>
              Downloads will show here
            </p>
          ) : (
            <div className="mt-4 space-y-3">
              <p className="text-[15px] font-semibold" style={{ color: jellyfinAccent }}>
                {torrents.some((t) => t.status === 'downloading') ? 'Now downloading' : 'Downloads will show here'}
              </p>
              {torrents.map((torrent) => (
                <div key={torrent.hash}>
                  <p className="mb-1 option-title truncate">{torrent.name}</p>
                  {torrent.status === 'downloading' && (
                    <p className="mb-1.5 option-subtitle" style={{ maxHeight: 'none', opacity: 1, transform: 'none', marginTop: '2px' }}>
                      &#8595;&nbsp;{torrent.speed_download_mbps ?? 0} mb/s&nbsp;&nbsp;&#8593;&nbsp;{torrent.speed_upload_mbps ?? 0} mb/s
                    </p>
                  )}
                  <div
                    className="w-full h-11 rounded-md flex items-center justify-center overflow-hidden relative"
                    style={{ borderColor: jellyfinAccent, backgroundColor: '#ffffff', border: `3px solid ${jellyfinAccent}` }}
                  >
                    {torrent.status === 'downloading' ? (
                      <div
                        className="absolute left-0 top-0 h-full"
                        style={{
                          width: `${torrent.progress_percentage ?? 0}%`,
                          backgroundColor: jellyfinAccent,
                          transition: 'width 800ms ease',
                        }}
                      />
                    ) : torrent.status === 'paused' ? (
                      <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24" style={{ color: jellyfinAccent }}>
                        <path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z" />
                      </svg>
                    ) : (
                      // unknown / queued — clock icon
                      <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={1.8} viewBox="0 0 24 24" style={{ color: jellyfinAccent }}>
                        <circle cx="12" cy="12" r="9" />
                        <path strokeLinecap="round" strokeLinejoin="round" d="M12 7v5l3 3" />
                      </svg>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </form>
      </div>
    </div>
  )
}
