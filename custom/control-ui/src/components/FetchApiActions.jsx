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
  const jellyfinAccent = '#8b5cf6'
  const jellyfinAccentRgb = '139, 92, 246'
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
    if (stateName === 'pending') return 'Adding'
    if (stateName === 'check') return 'Added'
    return 'Add'
  }

  const formatEta = (minutes) => {
    if (!minutes || minutes <= 0 || minutes >= 144000) return null
    if (minutes < 1) return 'Less than a minute'
    if (minutes >= 60) return `${Math.round(minutes / 60)} hrs`
    return `${minutes} mins`
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
        {/* Header */}
        <div className="mb-5 flex items-center justify-between group">
          <div className="flex items-center gap-3">
            <a
              href={jellyfinUrl}
              target="_blank"
              rel="noreferrer"
              aria-label="Open Jellyfin"
              className="w-9 h-9 rounded-full flex items-center justify-center flex-shrink-0"
              style={{ backgroundColor: `rgba(${jellyfinAccentRgb}, 0.12)` }}
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
            <h3 className="text-xl font-semibold text-gray-900">Add to Jellyfin</h3>
          </div>
          <div className="flex items-center gap-1">
            <a
              href={qbittorrentUrl}
              target="_blank"
              rel="noreferrer"
              aria-label="Open qBittorrent"
              className="inline-flex items-center justify-center w-8 h-8 rounded-md text-gray-400 hover:bg-gray-100 transition-colors"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v2a2 2 0 002 2h12a2 2 0 002-2v-2M7 10l5 5 5-5M12 15V3" />
              </svg>
            </a>

          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          {/* Input */}
          <div>
            <label className="block text-[10px] font-bold uppercase tracking-widest mb-2" style={{ color: jellyfinAccent }}>Torrent</label>
            <input
              type="text"
              value={movieName}
              disabled={isSubmitting}
              onChange={(e) => {
                setMovieName(e.target.value)
                if (urlError) setUrlError('')
              }}
              className="flat-input"
              style={urlError ? { borderBottomColor: jellyfinAccent } : undefined}
              placeholder="Magnet or url"
            />
            {urlError && <p className="mt-2 text-sm font-semibold" style={{ color: jellyfinAccent }}>{urlError}</p>}
          </div>

          {/* Settings collapsible */}
          <div>
            <button
              type="button"
              disabled={isSubmitting}
              onClick={() => setShowOptions((prev) => !prev)}
              className="inline-flex items-center gap-2 text-sm font-semibold text-gray-500 disabled:opacity-60 disabled:cursor-not-allowed hover:text-gray-700 transition-colors"
              aria-expanded={showOptions}
              aria-controls="import-options"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              <span>Options</span>
              <svg
                className={`w-4 h-4 transition-transform duration-220 ${showOptions ? 'rotate-180' : 'rotate-0'}`}
                fill="none" stroke="currentColor" viewBox="0 0 24 24"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M6 9l6 6 6-6" />
              </svg>
            </button>
            <div id="import-options" className={`options-panel ${showOptions ? 'options-panel-open' : ''}`}>
              <div className="options-panel-inner">
                <div className="pt-5 space-y-5">
                <div>
                  <p className="mb-1 text-sm font-medium text-gray-700">Save Location</p>
                  <input
                    type="text"
                    value={saveLocation}
                    disabled={isSubmitting}
                    onChange={(e) => setSaveLocation(e.target.value)}
                    className="flat-input"
                    placeholder={DEFAULT_SAVE_LOCATION}
                  />
                </div>
                <div>
                  <p className="mb-1 text-sm font-medium text-gray-700">Category</p>
                  <input
                    type="text"
                    value={qbittorrentCategory}
                    disabled={isSubmitting}
                    onChange={(e) => setQbittorrentCategory(e.target.value)}
                    className="flat-input"
                    placeholder={DEFAULT_CATEGORY}
                  />
                </div>
                <div>
                  <p className="mb-1 text-sm font-medium text-gray-700">Tags</p>
                  <input
                    type="text"
                    value={qbittorrentTags}
                    disabled={isSubmitting}
                    onChange={(e) => setQbittorrentTags(e.target.value)}
                    className="flat-input"
                    placeholder={DEFAULT_TAGS_PLACEHOLDER}
                  />
                </div>
                </div>
              </div>
            </div>
          </div>

          {/* Toggle options */}
          <div style={{ '--option-accent': jellyfinAccent }}>
            <div className="flex items-center justify-between py-3">
              <div>
                <p className="text-sm font-bold text-gray-900">Notify</p>
                <p key={`notify-${notify}`} className={`toggle-subtext text-xs font-semibold mt-0.5 ${notify ? '' : 'text-gray-400'}`} style={notify ? { color: jellyfinAccent } : undefined}>
                  {notify ? "Notify me in Slack when it's ready" : "Don't send me notifications"}
                </p>
              </div>
              <button
                type="button"
                role="switch"
                disabled={isSubmitting}
                aria-checked={notify}
                aria-label="Notify"
                onClick={() => setNotify((prev) => !prev)}
                className="toggle-switch"
              >
                <span className="toggle-thumb" />
              </button>
            </div>
            <div className="flex items-center justify-between py-3">
              <div>
                <p className="text-sm font-bold text-gray-900">Subtitles</p>
                <p key={`subs-${findSubs}`} className={`toggle-subtext text-xs font-semibold mt-0.5 ${findSubs ? '' : 'text-gray-400'}`} style={findSubs ? { color: jellyfinAccent } : undefined}>
                  {findSubs ? 'Try to find subtitles' : "Don't search for subtitles"}
                </p>
              </div>
              <button
                type="button"
                role="switch"
                disabled={isSubmitting}
                aria-checked={findSubs}
                aria-label="Subtitles"
                onClick={() => setFindSubs((prev) => !prev)}
                className="toggle-switch"
              >
                <span className="toggle-thumb" />
              </button>
            </div>
          </div>

          {/* Submit button */}
          <button
            type="submit"
            disabled={buttonState === 'pending'}
            className="import-action-btn relative w-full overflow-hidden rounded-lg font-semibold py-3 px-4 flex items-center justify-center transition-all duration-300 text-white disabled:cursor-not-allowed disabled:opacity-70"
            style={{ backgroundColor: jellyfinAccent }}
          >
            <span className="relative z-10 inline-flex items-center justify-center">
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

          {/* JSON log */}
          <div className="w-full rounded-lg bg-[#0d0d0d] p-4 min-h-[180px] border border-gray-800">
            <p className="text-[10px] font-bold uppercase tracking-widest text-gray-500 mb-2">Response</p>
            <pre
              className="text-[12px] leading-5 whitespace-pre-wrap break-all text-slate-300"
              dangerouslySetInnerHTML={{ __html: syntaxHighlightJson(jsonText) }}
            />
          </div>

          {/* Active Downloads */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <span
                key={torrents.length === 0 ? 'waiting' : 'active'}
                className="toggle-subtext text-[10px] font-bold uppercase tracking-widest"
                style={{ color: jellyfinAccent }}
              >
                {torrents.length === 0 ? 'Waiting for downloads' : 'Active Downloads'}
              </span>
            </div>
            {torrents.length > 0 && (
              <div>
                {torrents.map((torrent, idx) => {
                  const eta = formatEta(torrent.eta_minutes)
                  const progress = torrent.progress_percentage ?? 0
                  const status = torrent.status
                  const isDownloading = status === 'downloading'
                  const isPaused = status === 'paused'
                  const isUnknown = status === 'unknown'
                  const isError = status === 'error'
                  const barColor = isError ? '#ef4444' : (isPaused || isUnknown) ? '#9ca3af' : jellyfinAccent
                  return (
                    <div key={torrent.hash} className={idx < torrents.length - 1 ? 'mb-5' : ''}>
                      {/* Name + speed */}
                      <div className="flex items-center justify-between gap-3 mb-1.5">
                        <span className="text-sm font-semibold text-gray-800 truncate">{torrent.name}</span>
                        {isDownloading && (torrent.speed_download_mbps > 0 || torrent.speed_upload_mbps > 0) && (
                          <span
                            key={`${torrent.hash}-spd-${torrent.speed_download_mbps}-${torrent.speed_upload_mbps}`}
                            className="toggle-subtext text-[11px] font-semibold whitespace-nowrap flex-shrink-0"
                            style={{ color: jellyfinAccent }}
                          >
                            {torrent.speed_download_mbps > 0 && <>&#8595; {torrent.speed_download_mbps} mb/s</>}
                            {torrent.speed_download_mbps > 0 && torrent.speed_upload_mbps > 0 && <>&nbsp;&nbsp;</>}
                            {torrent.speed_upload_mbps > 0 && <>&#8593; {torrent.speed_upload_mbps} mb/s</>}
                          </span>
                        )}
                      </div>
                      {/* Progress bar */}
                      <div className="h-1 w-full rounded-full bg-gray-100 overflow-hidden">
                        <div
                          className="h-full rounded-full"
                          style={{
                            width: `${progress}%`,
                            backgroundColor: barColor,
                            transition: 'width 800ms ease, background-color 400ms ease',
                          }}
                        />
                      </div>
                      {/* Bottom row: seeders/leechers or status text left, ETA right */}
                      <div className="flex items-center justify-between mt-1">
                        <div>
                          {isDownloading && (torrent.seeders > 0 || torrent.leechers > 0) && (
                            <span
                              key={`${torrent.hash}-sl-${torrent.seeders}-${torrent.leechers}`}
                              className="toggle-subtext text-[11px] font-semibold flex items-center gap-1"
                              style={{ color: jellyfinAccent }}
                            >
                              {torrent.seeders > 0 && (
                                <>
                                  <svg className="w-3 h-3" fill="none" stroke="currentColor" strokeWidth={2.2} viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 19V5m0 0-5 5m5-5 5 5" />
                                  </svg>
                                  <span>{torrent.seeders}</span>
                                </>
                              )}
                              {torrent.leechers > 0 && (
                                <>
                                  <svg className="w-3 h-3" fill="none" stroke="currentColor" strokeWidth={2.2} viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 5v14m0 0-5-5m5 5 5-5" />
                                  </svg>
                                  <span>{torrent.leechers}</span>
                                </>
                              )}
                            </span>
                          )}
                          {isPaused && (
                            <span key="paused" className="toggle-subtext text-[11px] font-semibold text-gray-400">Paused</span>
                          )}
                          {isUnknown && (
                            <span key="unknown" className="toggle-subtext text-[11px] font-semibold text-red-500">Unknown status</span>
                          )}
                          {isError && (
                            <span key="error" className="toggle-subtext text-[11px] font-semibold text-red-500">Error while trying to download</span>
                          )}
                        </div>
                        {isDownloading && eta && (
                          <span
                            key={`${torrent.hash}-eta-${eta}`}
                            className="toggle-subtext text-[11px] font-semibold whitespace-nowrap flex items-center gap-1 flex-shrink-0"
                            style={{ color: jellyfinAccent }}
                          >
                            <svg className="w-3 h-3" fill="none" stroke="currentColor" strokeWidth={1.8} viewBox="0 0 24 24">
                              <circle cx="12" cy="12" r="9" />
                              <path strokeLinecap="round" strokeLinejoin="round" d="M12 7v5l3 3" />
                            </svg>
                            {eta}
                          </span>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </form>
      </div>
    </div>
  )
}
