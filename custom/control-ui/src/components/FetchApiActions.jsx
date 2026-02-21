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
  const [urlErrorKey, setUrlErrorKey] = useState(0)
  const [jsonText, setJsonText] = useState('{}')
  const [jsonKey, setJsonKey] = useState(0)
  const [buttonState, setButtonState] = useState('idle') // idle | pending
  const [buttonIcon, setButtonIcon] = useState('arrows') // arrows | pending | check
  const [iconTransition, setIconTransition] = useState(null)
  const [torrents, setTorrents] = useState([])
  const [hasItems, setHasItems] = useState(false)
  const [exitingTorrents, setExitingTorrents] = useState([])
  const [enteringHashes, setEnteringHashes] = useState(new Set())
  const timersRef = useRef([])
  const iconFlowRef = useRef(0)
  const buttonIconRef = useRef('arrows')
  const prevItemsRef = useRef([])
  const isFirstFetchRef = useRef(true)
  const isSubmitting = buttonState === 'pending'
  const [hoveredSubsHash, setHoveredSubsHash] = useState(null)
  const [subsTextHoveredHash, setSubsTextHoveredHash] = useState(null)
  const [hoveredNameHash, setHoveredNameHash] = useState(null)
  const [inputAnimPhase, setInputAnimPhase] = useState(false)
  const [inputShake, setInputShake] = useState(false)
  const [inputApiError, setInputApiError] = useState(false)
  const [queryMode, setQueryMode] = useState(false)
  const [queryModeKey, setQueryModeKey] = useState(0)

  useEffect(() => {
    return () => {
      timersRef.current.forEach(clearTimeout)
      timersRef.current = []
    }
  }, [])

  const ENTER_MS = 900
  const EXIT_MS = 900

  useEffect(() => {
    const fetchTorrents = async () => {
      try {
        const data = await getTorrents()
        const items = data.items || []
        if (!isFirstFetchRef.current) {
          const currentHashes = new Set(items.map(t => t.hash))
          const prevHashes = new Set(prevItemsRef.current.map(t => t.hash))
          // Entering hashes
          const entering = new Set([...currentHashes].filter(h => !prevHashes.has(h)))
          if (entering.size > 0) {
            setEnteringHashes(prev => new Set([...prev, ...entering]))
            setTimeout(() => setEnteringHashes(prev => {
              const next = new Set(prev)
              entering.forEach(h => next.delete(h))
              return next
            }), ENTER_MS + 50)
          }
          // Exiting items
          const exitItems = prevItemsRef.current.filter(t => !currentHashes.has(t.hash))
          if (exitItems.length > 0) {
            const exitHashes = new Set(exitItems.map(t => t.hash))
            setExitingTorrents(prev => [
              ...prev.filter(t => !exitHashes.has(t.hash)),
              ...exitItems,
            ])
            setTimeout(() => {
              setExitingTorrents(prev => prev.filter(t => !exitHashes.has(t.hash)))
            }, EXIT_MS + 50)
          }
        }
        isFirstFetchRef.current = false
        prevItemsRef.current = items
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

  const setJsonOutput = (data) => {
    setJsonText(JSON.stringify(data ?? {}, null, 2))
    setJsonKey(k => k + 1)
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
        className="block flex-shrink-0"
        style={{
          width: '15px',
          height: '15px',
          backgroundColor: 'currentColor',
          WebkitMaskImage: 'url(https://i.imgur.com/lrTz6dE.png)',
          maskImage: 'url(https://i.imgur.com/lrTz6dE.png)',
          WebkitMaskSize: 'contain',
          maskSize: 'contain',
          WebkitMaskRepeat: 'no-repeat',
          maskRepeat: 'no-repeat',
          WebkitMaskPosition: 'center',
          maskPosition: 'center',
          filter: 'drop-shadow(0 0 0.6px currentColor) drop-shadow(0 0 0.6px currentColor)',
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
    if (minutes <= 1) return 'Around a minute'
    if (minutes >= 60) return `${Math.round(minutes / 60)} hrs`
    return `${Math.round(minutes)} mins`
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (buttonState === 'pending') {
      return
    }
    const value = movieName.trim()

    if (!value) {
      setUrlError('* Torrent URL is required')
      setUrlErrorKey((k) => k + 1)
      setJsonText('{}')
      return
    }

    setUrlError('')
    setInputApiError(false)
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
      setJsonOutput(response)
      requestSucceeded = true
    } catch (error) {
      setJsonOutput({ error: error.message || 'Request failed' })
    }

    if (!requestSucceeded) {
      setInputApiError(true)
      setInputShake(true)
      const shakeTimer = setTimeout(() => setInputShake(false), 500)
      timersRef.current.push(shakeTimer)
      await transitionButtonIcon('arrows', flowId)
      if (flowId === iconFlowRef.current) {
        setButtonState('idle')
      }
      return
    }

    setInputAnimPhase(true)
    const animTimer = setTimeout(() => { setMovieName(''); setInputAnimPhase(false) }, 1750)
    timersRef.current.push(animTimer)

    await transitionButtonIcon('check', flowId)
    if (flowId === iconFlowRef.current) {
      setButtonState('idle')
    }
    await wait(SUCCESS_ICON_HOLD_MS)
    await transitionButtonIcon('arrows', flowId)
  }

  const displayTorrents = [
    ...torrents,
    ...exitingTorrents.filter(et => !torrents.find(t => t.hash === et.hash)),
  ]

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
              style={{ backgroundColor: 'rgba(0, 128, 255, 0.12)' }}
            >
              <span
                aria-hidden="true"
                className="block w-4 h-4"
                style={{
                  backgroundColor: '#0080ff',
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
              className="inline-flex items-center justify-center w-8 h-8 rounded-md hover:bg-gray-100 transition-colors"
              style={{ color: '#0080ff' }}
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
            <label key={`label-${queryModeKey}`} className="toggle-subtext block text-[10px] font-bold uppercase tracking-widest mb-2" style={{ color: jellyfinAccent }}>
              {queryMode ? 'Query' : 'Torrent'}
            </label>
            <div className={`flat-input-wrap${inputShake ? ' flat-input-wrap-shake' : ''}`}>
              <input
                type="text"
                value={movieName}
                disabled={isSubmitting}
                onChange={(e) => {
                  setMovieName(e.target.value)
                  if (urlError) setUrlError('')
                  if (inputApiError) setInputApiError(false)
                }}
                className={`flat-input flat-input-no-placeholder${inputAnimPhase ? ' flat-input-text-out' : ''}`}
                placeholder=""
              />
              {movieName === '' && (
                <span key={`ph-${queryModeKey}`} className="fake-placeholder toggle-subtext">
                  {queryMode ? 'Name or keyword of movie or show' : 'Magnet or url'}
                </span>
              )}
              <div className={`flat-input-line${urlError ? ' flat-input-line-error' : ''}${inputApiError ? ' flat-input-line-api-error' : ''}${inputAnimPhase ? ' flat-input-line-anim' : ''}`} />
            </div>
            {urlError && <p key={urlErrorKey} className="toggle-subtext mt-2 text-xs font-semibold" style={{ color: jellyfinAccent }}>{urlError}</p>}
          </div>

          {/* Toggle options */}
          <div style={{ '--option-accent': jellyfinAccent }}>
            {/* Query */}
            <div className="flex items-center justify-between py-3">
              <div className="flex items-start gap-3">
                <span
                  aria-hidden="true"
                  className="mt-1 block w-5 h-5 flex-shrink-0 transition-colors duration-300"
                  style={{
                    backgroundColor: queryMode ? '#111827' : '#9ca3af',
                    WebkitMaskImage: 'url(https://i.imgur.com/vNR0MzV.png)',
                    maskImage: 'url(https://i.imgur.com/vNR0MzV.png)',
                    WebkitMaskSize: 'contain',
                    maskSize: 'contain',
                    WebkitMaskRepeat: 'no-repeat',
                    maskRepeat: 'no-repeat',
                    WebkitMaskPosition: 'center',
                    maskPosition: 'center',
                  }}
                />
                <div>
                  <p className="text-sm font-bold text-gray-900 cursor-pointer select-none" onClick={() => { setQueryMode(p => !p); setQueryModeKey(k => k + 1) }}>Query</p>
                  <p key={`query-${queryMode}`} className={`toggle-subtext text-xs font-semibold mt-0.5 cursor-pointer select-none ${queryMode ? '' : 'text-gray-400'}`} style={queryMode ? { color: jellyfinAccent } : undefined} onClick={() => { setQueryMode(p => !p); setQueryModeKey(k => k + 1) }}>
                    {queryMode ? 'Search for movie or show' : 'Use torrent url'}
                  </p>
                </div>
              </div>
              <button
                type="button"
                role="switch"
                disabled={isSubmitting}
                aria-checked={queryMode}
                aria-label="Query"
                onClick={() => { setQueryMode(p => !p); setQueryModeKey(k => k + 1) }}
                className="toggle-switch"
              >
                <span className="toggle-thumb" />
              </button>
            </div>
            {/* Notify */}
            <div className="flex items-center justify-between py-3">
              <div className="flex items-start gap-3">
                <span
                  aria-hidden="true"
                  className="mt-1 block w-5 h-5 flex-shrink-0 transition-colors duration-300"
                  style={{
                    backgroundColor: notify ? '#111827' : '#9ca3af',
                    WebkitMaskImage: 'url(https://i.imgur.com/Z2E0Nz2.png)',
                    maskImage: 'url(https://i.imgur.com/Z2E0Nz2.png)',
                    WebkitMaskSize: 'contain',
                    maskSize: 'contain',
                    WebkitMaskRepeat: 'no-repeat',
                    maskRepeat: 'no-repeat',
                    WebkitMaskPosition: 'center',
                    maskPosition: 'center',
                  }}
                />
                <div>
                  <p className="text-sm font-bold text-gray-900 cursor-pointer select-none" onClick={() => setNotify(prev => !prev)}>Notify</p>
                  <p key={`notify-${notify}`} className={`toggle-subtext text-xs font-semibold mt-0.5 cursor-pointer select-none ${notify ? '' : 'text-gray-400'}`} style={notify ? { color: jellyfinAccent } : undefined} onClick={() => setNotify(prev => !prev)}>
                    {notify ? "Notify me in Slack when it's ready" : "Don't send me notifications"}
                  </p>
                </div>
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
              <div className="flex items-start gap-3">
                <span
                  aria-hidden="true"
                  className="mt-1 block w-5 h-5 flex-shrink-0 transition-colors duration-300"
                  style={{
                    backgroundColor: findSubs ? '#111827' : '#9ca3af',
                    WebkitMaskImage: 'url(https://i.imgur.com/2SzFid0.png)',
                    maskImage: 'url(https://i.imgur.com/2SzFid0.png)',
                    WebkitMaskSize: 'contain',
                    maskSize: 'contain',
                    WebkitMaskRepeat: 'no-repeat',
                    maskRepeat: 'no-repeat',
                    WebkitMaskPosition: 'center',
                    maskPosition: 'center',
                  }}
                />
                <div>
                  <p className="text-sm font-bold text-gray-900 cursor-pointer select-none" onClick={() => setFindSubs(prev => !prev)}>Subtitles</p>
                  <p key={`subs-${findSubs}`} className={`toggle-subtext text-xs font-semibold mt-0.5 cursor-pointer select-none ${findSubs ? '' : 'text-gray-400'}`} style={findSubs ? { color: jellyfinAccent } : undefined} onClick={() => setFindSubs(prev => !prev)}>
                    {findSubs ? 'Search and download subtitles' : "Don't search for subtitles"}
                  </p>
                </div>
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
                  <p className="mb-0.5 text-sm font-medium text-gray-700">Save Location</p>
                  <div className="flat-input-wrap">
                    <input
                      type="text"
                      value={saveLocation}
                      disabled={isSubmitting}
                      onChange={(e) => setSaveLocation(e.target.value)}
                      className="flat-input"
                      placeholder={DEFAULT_SAVE_LOCATION}
                    />
                    <div className="flat-input-line" />
                  </div>
                </div>
                <div>
                  <p className="mb-0.5 text-sm font-medium text-gray-700">Category</p>
                  <div className="flat-input-wrap">
                    <input
                      type="text"
                      value={qbittorrentCategory}
                      disabled={isSubmitting}
                      onChange={(e) => setQbittorrentCategory(e.target.value)}
                      className="flat-input"
                      placeholder={DEFAULT_CATEGORY}
                    />
                    <div className="flat-input-line" />
                  </div>
                </div>
                <div>
                  <p className="mb-0.5 text-sm font-medium text-gray-700">Tags</p>
                  <div className="flat-input-wrap">
                    <input
                      type="text"
                      value={qbittorrentTags}
                      disabled={isSubmitting}
                      onChange={(e) => setQbittorrentTags(e.target.value)}
                      className="flat-input"
                      placeholder={DEFAULT_TAGS_PLACEHOLDER}
                    />
                    <div className="flat-input-line" />
                  </div>
                </div>
                </div>
              </div>
            </div>
          </div>

          {/* Submit button */}
          <div className={`submit-btn-panel${queryMode ? ' submit-btn-panel-hidden' : ''}`}>
            <div className="submit-btn-panel-inner">
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
            </div>
          </div>

          {/* JSON log */}
          <div className="w-full rounded-lg bg-[#0d0d0d] p-4 min-h-[180px] border border-gray-800">
            <p className="text-[10px] font-bold uppercase tracking-widest text-gray-500 mb-2">Response</p>
            <pre
              key={jsonKey}
              className="json-fade text-[12px] leading-5 whitespace-pre-wrap break-all text-slate-300"
              dangerouslySetInnerHTML={{ __html: syntaxHighlightJson(jsonText) }}
            />
          </div>

          {/* Active Downloads */}
          <div className="pt-2">
            <div className="flex items-center justify-between mb-3">
              <span
                key={torrents.map(t => t.hash).sort().join(',') || 'empty'}
                className="toggle-subtext text-[10px] font-bold uppercase tracking-widest"
                style={{ color: torrents.some(t => t.status === 'downloading') ? jellyfinAccent : '#9ca3af' }}
              >
                {torrents.some(t => t.status === 'downloading') ? 'Active Downloads' : 'Waiting for downloads'}
              </span>
            </div>
            {displayTorrents.length > 0 && (
              <div>
                {displayTorrents.map((torrent, idx) => {
                  const eta = formatEta(torrent.eta_minutes)
                  const progress = torrent.progress_percentage ?? 0
                  const status = torrent.status
                  const isDownloading = status === 'downloading'
                  const isPaused = status === 'paused'
                  const isUnknown = status === 'unknown'
                  const isError = status === 'error'
                  const isCompleted = status === 'completed'
                  const barColor = isError ? '#ef4444' : (isPaused || isUnknown) ? '#9ca3af' : jellyfinAccent
                  const isEntering = enteringHashes.has(torrent.hash)
                  const isExiting = exitingTorrents.some(et => et.hash === torrent.hash)
                  const animClass = isEntering ? 'torrent-item-enter' : isExiting ? 'torrent-item-exit' : 'torrent-item-outer'

                  // Subtitles tag logic
                  const tags = torrent.tags || []
                  const findSubsTag = tags.find(t => typeof t === 'string' && t.startsWith('jellyfin:find_subs='))
                  const isSubsHovered = hoveredSubsHash === torrent.hash
                  const isSubsTextHovered = subsTextHoveredHash === torrent.hash

                  let subsIconColor, subsIconOpacity, subsClickable
                  let subsTextA = null, subsTextAColor, subsTextB = null, subsTextBColor

                  if (isUnknown || isError) {
                    subsIconColor = '#9ca3af'; subsIconOpacity = 1; subsClickable = false
                  } else if (isPaused) {
                    subsIconColor = '#9ca3af'; subsIconOpacity = 1; subsClickable = true
                    if (findSubsTag === 'jellyfin:find_subs=pending') {
                      subsTextA = 'Will download subtitles'; subsTextAColor = '#111827'
                      subsTextB = 'Cancel'; subsTextBColor = '#e00000'
                    } else if (findSubsTag === 'jellyfin:find_subs=completed') {
                      subsTextA = 'Subtitles downloaded'; subsTextAColor = '#111827'
                    } else if (findSubsTag === 'jellyfin:find_subs=partially_completed') {
                      subsTextA = 'Subtitles may not be present in all episodes/movies'; subsTextAColor = '#111827'
                    } else if (findSubsTag === 'jellyfin:find_subs=already_present') {
                      subsTextA = 'Subtitles included with torrent'; subsTextAColor = '#111827'
                    } else if (findSubsTag === 'jellyfin:find_subs=failed') {
                      subsTextA = 'Failed to find and/or download subtitles'; subsTextAColor = '#e00000'
                    } else {
                      subsTextA = 'Will not download subtitles'; subsTextAColor = '#111827'
                      subsTextB = 'Download'; subsTextBColor = jellyfinAccent
                    }
                  } else if (isDownloading || isCompleted) {
                    subsClickable = true
                    if (findSubsTag === 'jellyfin:find_subs=pending') {
                      subsIconColor = jellyfinAccent; subsIconOpacity = 1
                      subsTextA = 'Will download subtitles'; subsTextAColor = jellyfinAccent
                      subsTextB = 'Cancel'; subsTextBColor = '#e00000'
                    } else if (findSubsTag === 'jellyfin:find_subs=completed') {
                      subsIconColor = jellyfinAccent; subsIconOpacity = 1
                      subsTextA = 'Subtitles downloaded'; subsTextAColor = '#1DB954'
                    } else if (findSubsTag === 'jellyfin:find_subs=partially_completed') {
                      subsIconColor = jellyfinAccent; subsIconOpacity = 1
                      subsTextA = 'Subtitles may not be present in all episodes/movies'; subsTextAColor = jellyfinAccent
                    } else if (findSubsTag === 'jellyfin:find_subs=already_present') {
                      subsIconColor = jellyfinAccent; subsIconOpacity = 1
                      subsTextA = 'Subtitles already included with torrent'; subsTextAColor = '#1DB954'
                    } else if (findSubsTag === 'jellyfin:find_subs=failed') {
                      subsIconColor = '#e00000'; subsIconOpacity = 1
                      subsTextA = 'Failed to find and/or download subtitles'; subsTextAColor = '#e00000'
                    } else {
                      subsIconColor = '#9ca3af'; subsIconOpacity = isSubsHovered ? 1 : 0.5
                      subsTextA = 'Will not download subtitles'; subsTextAColor = '#111827'
                      subsTextB = 'Download'; subsTextBColor = jellyfinAccent
                    }
                  } else {
                    subsIconColor = '#9ca3af'; subsIconOpacity = 1; subsClickable = false
                  }

                  const isNameHovered = hoveredNameHash === torrent.hash

                  return (
                    <div key={torrent.hash} className={animClass}>
                      <div className={`torrent-item-inner${idx < displayTorrents.length - 1 ? ' pb-[18px]' : ''}`}>
                        {/* Name + icons */}
                        <div className="flex items-center justify-between gap-3 mb-1">
                          <div
                            className={`name-clip${isNameHovered ? ' name-clip-expanded' : ''}`}
                            onMouseEnter={() => setHoveredNameHash(torrent.hash)}
                            onMouseLeave={() => setHoveredNameHash(null)}
                          >
                            <span className="text-sm font-bold text-gray-900 whitespace-nowrap">{torrent.name}</span>
                          </div>
                          {/* Subtitle icon + gear icon */}
                          <div className="flex items-center gap-2 flex-shrink-0">
                            {/* Subtitle section: row-reverse so icon stays right, text slides out left */}
                            <div
                              className="flex flex-row-reverse items-center gap-1"
                              style={{ cursor: subsClickable ? 'pointer' : 'default' }}
                              onMouseEnter={() => { if (subsClickable) setHoveredSubsHash(torrent.hash) }}
                              onMouseLeave={() => { setHoveredSubsHash(null); setSubsTextHoveredHash(null) }}
                            >
                              {/* Icon */}
                              <span
                                aria-hidden="true"
                                className="block flex-shrink-0"
                                style={{
                                  width: '22px',
                                  height: '22px',
                                  backgroundColor: subsIconColor,
                                  opacity: subsIconOpacity,
                                  transition: 'background-color 200ms ease, opacity 200ms ease',
                                  WebkitMaskImage: 'url(https://i.imgur.com/2SzFid0.png)',
                                  maskImage: 'url(https://i.imgur.com/2SzFid0.png)',
                                  WebkitMaskSize: 'contain',
                                  maskSize: 'contain',
                                  WebkitMaskRepeat: 'no-repeat',
                                  maskRepeat: 'no-repeat',
                                  WebkitMaskPosition: 'center',
                                  maskPosition: 'center',
                                }}
                              />
                              {/* Text: slides in from icon (right) toward left */}
                              {subsTextA && (
                                <div
                                  className={`subs-text-clip${isSubsHovered ? ' subs-text-visible' : ''}`}
                                  onMouseEnter={() => { if (subsTextB) setSubsTextHoveredHash(torrent.hash) }}
                                  onMouseLeave={() => setSubsTextHoveredHash(null)}
                                >
                                  {isSubsTextHovered && subsTextB ? (
                                    <span
                                      key={`${torrent.hash}-subs-b`}
                                      className="subs-text-swap subs-bubble"
                                      style={{ backgroundColor: subsTextBColor, fontSize: '15px' }}
                                    >
                                      {subsTextB}
                                    </span>
                                  ) : (
                                    <span
                                      key={`${torrent.hash}-subs-a`}
                                      className="subs-bubble"
                                      style={{ backgroundColor: subsTextAColor, fontSize: '15px' }}
                                    >
                                      {subsTextA}
                                    </span>
                                  )}
                                </div>
                              )}
                            </div>
                            {/* Gear icon */}
                            <svg
                              aria-hidden="true"
                              style={{ width: '22px', height: '22px', color: '#9ca3af', flexShrink: 0 }}
                              fill="none" stroke="currentColor" strokeWidth={2.5} viewBox="0 0 24 24"
                            >
                              <path strokeLinecap="round" strokeLinejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                              <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                            </svg>
                          </div>
                        </div>
                        {/* Progress bar */}
                        <div className="h-1 w-full rounded-full bg-gray-100 overflow-hidden">
                          <div
                            className={`h-full rounded-full${isDownloading ? ' progress-bar-gradient' : ''}`}
                            style={{
                              width: `${progress}%`,
                              ...(!isDownloading ? { backgroundColor: isCompleted ? '#1DB954' : barColor } : {}),
                              transition: 'width 800ms ease, background-color 400ms ease',
                            }}
                          />
                        </div>
                        {/* Bottom row */}
                        <div className="flex items-center justify-between mt-1">
                          {/* Left: ETA or status text */}
                          <div>
                            {isDownloading && eta && (
                              <span className="text-xs font-semibold whitespace-nowrap" style={{ color: jellyfinAccent }}>
                                <span key={`${torrent.hash}-eta-${eta}`} className="stat-value">{eta}</span>
                              </span>
                            )}
                            {isPaused && (
                              <span key="paused" className="toggle-subtext text-xs font-semibold text-gray-400">Paused</span>
                            )}
                            {isUnknown && (
                              <span key="unknown" className="toggle-subtext text-xs font-semibold text-red-500">Unknown status</span>
                            )}
                            {isError && (
                              <span key="error" className="toggle-subtext text-xs font-semibold text-red-500">Torrent error</span>
                            )}
                            {isCompleted && (
                              <span key="completed" className="toggle-subtext text-xs font-semibold" style={{ color: '#1DB954' }}>Completed</span>
                            )}
                          </div>
                          {/* Right: down/up speed */}
                          {isDownloading && (torrent.speed_download_mbps > 0 || torrent.speed_upload_mbps > 0) && (
                            <span className="text-xs font-semibold whitespace-nowrap flex-shrink-0 flex items-center gap-1" style={{ color: jellyfinAccent }}>
                              {torrent.speed_download_mbps > 0 && (
                                <>
                                  &#8595;&nbsp;<span key={`${torrent.hash}-dl-${torrent.speed_download_mbps}`} className="stat-value">{torrent.speed_download_mbps} mb/s</span>
                                </>
                              )}
                              {torrent.speed_download_mbps > 0 && torrent.speed_upload_mbps > 0 && <span>&nbsp;&nbsp;</span>}
                              {torrent.speed_upload_mbps > 0 && (
                                <>
                                  &#8593;&nbsp;<span key={`${torrent.hash}-ul-${torrent.speed_upload_mbps}`} className="stat-value">{torrent.speed_upload_mbps} mb/s</span>
                                </>
                              )}
                            </span>
                          )}
                        </div>
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
