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
  const [expandedHashes, setExpandedHashes] = useState(new Set())
  const [expandTransitions, setExpandTransitions] = useState(new Map()) // hash → 'opening' | 'closing'
  const expandTimersRef = useRef({})

  useEffect(() => {
    return () => {
      timersRef.current.forEach(clearTimeout)
      timersRef.current = []
      Object.values(expandTimersRef.current).forEach(clearTimeout)
      expandTimersRef.current = {}
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

  const toggleExpand = (hash) => {
    const currentlyExpanded = expandedHashes.has(hash)
    const direction = currentlyExpanded ? 'closing' : 'opening'
    if (expandTimersRef.current[hash]) clearTimeout(expandTimersRef.current[hash])
    setExpandTransitions(prev => new Map(prev).set(hash, direction))
    expandTimersRef.current[hash] = setTimeout(() => {
      setExpandedHashes(prev => {
        const next = new Set(prev)
        if (direction === 'opening') next.add(hash)
        else next.delete(hash)
        return next
      })
      setExpandTransitions(prev => {
        const next = new Map(prev)
        next.delete(hash)
        return next
      })
      delete expandTimersRef.current[hash]
    }, 1200)
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
            {urlError && <p key={urlErrorKey} className="toggle-subtext mt-2 text-xs font-semibold" style={{ color: jellyfinAccent }}>{urlError}</p>}
          </div>

          {/* Toggle options */}
          <div style={{ '--option-accent': jellyfinAccent }}>
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
                  <p className="text-sm font-bold text-gray-900">Notify</p>
                  <p key={`notify-${notify}`} className={`toggle-subtext text-xs font-semibold mt-0.5 ${notify ? '' : 'text-gray-400'}`} style={notify ? { color: jellyfinAccent } : undefined}>
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
                  <p className="text-sm font-bold text-gray-900">Subtitles</p>
                  <p key={`subs-${findSubs}`} className={`toggle-subtext text-xs font-semibold mt-0.5 ${findSubs ? '' : 'text-gray-400'}`} style={findSubs ? { color: jellyfinAccent } : undefined}>
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
                style={{ color: torrents.length === 0 ? '#9ca3af' : jellyfinAccent }}
              >
                {torrents.length === 0 ? 'Waiting for downloads' : 'Active Downloads'}
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
                  const isExpanded = expandedHashes.has(torrent.hash)
                  const expandDir = expandTransitions.get(torrent.hash)
                  const iconActive = isExpanded ? expandDir !== 'closing' : expandDir === 'opening'
                  const showNormal = !isExpanded || expandDir === 'closing'
                  const showExpanded = isExpanded || expandDir === 'opening'
                  const normalClass = expandDir === 'opening' ? 'ti-normal-exit' : expandDir === 'closing' ? 'ti-normal-enter' : ''
                  const expandedClass = expandDir === 'opening' ? 'ti-expanded-enter' : expandDir === 'closing' ? 'ti-expanded-exit' : ''
                  return (
                    <div key={torrent.hash} className={animClass}>
                      <div className={`torrent-item-inner${idx < displayTorrents.length - 1 ? ' pb-5' : ''}`}>
                      <div className="flex gap-3">
                        {/* Left icon toggle button */}
                        <div className="flex items-center flex-shrink-0">
                          <button
                            onClick={() => toggleExpand(torrent.hash)}
                            className={`torrent-expand-btn${iconActive ? ' active' : ''}`}
                            aria-label="Toggle details"
                          >
                            <span
                              className="torrent-expand-icon"
                              style={{
                                display: 'block',
                                width: '16px',
                                height: '16px',
                                WebkitMaskImage: 'url(https://i.imgur.com/5LstlCU.png)',
                                maskImage: 'url(https://i.imgur.com/5LstlCU.png)',
                                WebkitMaskSize: 'contain',
                                maskSize: 'contain',
                                WebkitMaskRepeat: 'no-repeat',
                                maskRepeat: 'no-repeat',
                                WebkitMaskPosition: 'center',
                                maskPosition: 'center',
                              }}
                            />
                          </button>
                        </div>
                        {/* Content */}
                        <div className="flex-1 min-w-0" style={{ position: 'relative' }}>
                          {showNormal && (
                            <div className={`ti-normal${normalClass ? ` ${normalClass}` : ''}`}>
                              {/* Name + speeds row */}
                              <div className="ti-top-row flex items-center justify-between gap-3 mb-1.5">
                                <span className="text-sm font-semibold text-gray-800 truncate">{torrent.name}</span>
                                {isDownloading && (torrent.speed_download_mbps > 0 || torrent.speed_upload_mbps > 0) && (
                                  <span className="ti-speeds text-xs font-semibold whitespace-nowrap flex-shrink-0 flex items-center gap-1" style={{ color: jellyfinAccent }}>
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
                              {/* Progress bar */}
                              <div className="ti-progress-wrap h-1 w-full rounded-full bg-gray-100 overflow-hidden">
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
                              <div className="ti-bottom-row flex items-center justify-between mt-1">
                                <div>
                                  {isDownloading && (torrent.seeders > 0 || torrent.leechers > 0) && (
                                    <span className="text-xs font-semibold flex items-center gap-2" style={{ color: jellyfinAccent }}>
                                      {torrent.seeders > 0 && (
                                        <span className="flex items-center gap-0.5">
                                          <svg className="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" strokeWidth={2.2} viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" d="M12 19V5m0 0-5 5m5-5 5 5" />
                                          </svg>
                                          <span key={`${torrent.hash}-s-${torrent.seeders}`} className="stat-value">{torrent.seeders}</span>
                                        </span>
                                      )}
                                      {torrent.leechers > 0 && (
                                        <span className="flex items-center gap-0.5">
                                          <svg className="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" strokeWidth={2.2} viewBox="0 0 24 24">
                                            <path strokeLinecap="round" strokeLinejoin="round" d="M12 5v14m0 0-5-5m5 5 5-5" />
                                          </svg>
                                          <span key={`${torrent.hash}-l-${torrent.leechers}`} className="stat-value">{torrent.leechers}</span>
                                        </span>
                                      )}
                                    </span>
                                  )}
                                  {isPaused && (
                                    <span key="paused" className="toggle-subtext text-xs font-semibold text-gray-400 flex items-center">Paused</span>
                                  )}
                                  {isUnknown && (
                                    <span key="unknown" className="toggle-subtext text-xs font-semibold text-red-500 flex items-center">Unknown status</span>
                                  )}
                                  {isError && (
                                    <span key="error" className="toggle-subtext text-xs font-semibold text-red-500 flex items-center">Torrent error</span>
                                  )}
                                  {isCompleted && (
                                    <span key="completed" className="toggle-subtext text-xs font-semibold flex items-center" style={{ color: '#1DB954' }}>Completed</span>
                                  )}
                                </div>
                                {isDownloading && eta && (
                                  <span className="text-xs font-semibold whitespace-nowrap flex-shrink-0" style={{ color: jellyfinAccent }}>
                                    <span key={`${torrent.hash}-eta-${eta}`} className="stat-value">{eta}</span>
                                  </span>
                                )}
                              </div>
                            </div>
                          )}
                          {showExpanded && (
                            <div className={`ti-expanded${expandedClass ? ` ${expandedClass}` : ''}`}>
                              {/* Tag icon row */}
                              <div className="ti-tag-row mb-2">
                                <button
                                  className="inline-flex items-center justify-center rounded-md w-5 h-5 hover:bg-purple-50 transition-colors"
                                  style={{ color: jellyfinAccent }}
                                  aria-label="Tag"
                                >
                                  <svg className="w-4 h-4" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z" />
                                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 6h.008v.008H6V6z" />
                                  </svg>
                                </button>
                              </div>
                              {/* Name only */}
                              <div className="ti-name-only">
                                <span className="text-sm font-semibold text-gray-800 truncate block">{torrent.name}</span>
                              </div>
                            </div>
                          )}
                        </div>{/* end content */}
                      </div>{/* end flex gap-3 */}
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
