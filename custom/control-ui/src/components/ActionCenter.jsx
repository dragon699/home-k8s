import React, { useEffect, useMemo, useRef, useState } from 'react'
import ApplicationActions from './ApplicationActions'
import CustomActions from './CustomActions'
import FetchApiActions from './FetchApiActions'

function QbittorrentNavIcon() {
  return (
    <svg className="nav-item-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
      <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2M7 10l5 5 5-5M12 15V3" />
    </svg>
  )
}

function MaskNavIcon({ url }) {
  return (
    <span
      aria-hidden="true"
      className="nav-item-icon-mask"
      style={{
        WebkitMaskImage: `url(${url})`,
        maskImage: `url(${url})`,
      }}
    />
  )
}

export default function ActionCenter({ activeTab, setActiveTab }) {
  const tabs = useMemo(
    () => [
      {
        id: 'custom',
        primaryLabel: 'Panel',
        secondaryLabel: 'fetch-api',
        accent: { rgb: '205, 239, 60', ink: '#244133', secondary: '#7d9821' },
        icon: <MaskNavIcon url="https://i.imgur.com/wWCcoW6.png" />,
        component: <CustomActions />,
      },
      {
        id: 'fetch-api',
        primaryLabel: 'connector-downloader',
        secondaryLabel: 'fetch-api',
        accent: { rgb: '44, 116, 216', ink: '#1f63c2', secondary: '#3c84d7' },
        icon: <QbittorrentNavIcon />,
        component: <FetchApiActions />,
      },
      {
        id: 'overview',
        primaryLabel: 'connector-grafana',
        secondaryLabel: 'fetch-api',
        accent: { rgb: '243, 130, 32', ink: '#c75c07', secondary: '#d9731d' },
        icon: <MaskNavIcon url="https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/svg/grafana.svg" />,
        component: <ApplicationActions />,
      },
    ],
    []
  )
  const tabOrder = useMemo(() => tabs.map((tab) => tab.id), [tabs])
  const [displayedTab, setDisplayedTab] = useState(activeTab)
  const [outgoingTab, setOutgoingTab] = useState(null)
  const [transitionDirection, setTransitionDirection] = useState('forward')
  const timeoutRef = useRef(null)

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [])

  useEffect(() => {
    if (activeTab === displayedTab) {
      return
    }

    const currentIndex = tabOrder.indexOf(displayedTab)
    const nextIndex = tabOrder.indexOf(activeTab)
    const direction = nextIndex >= currentIndex ? 'forward' : 'backward'

    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current)
    }

    setTransitionDirection(direction)
    setOutgoingTab(displayedTab)
    setDisplayedTab(activeTab)

    timeoutRef.current = setTimeout(() => {
      setOutgoingTab(null)
      timeoutRef.current = null
    }, 500)
  }, [activeTab, displayedTab, tabOrder])

  const displayedComponent = tabs.find((tab) => tab.id === displayedTab)?.component ?? null
  const outgoingComponent = tabs.find((tab) => tab.id === outgoingTab)?.component ?? null
  const isTransitioning = outgoingTab !== null
  const activeAccent = tabs.find((tab) => tab.id === activeTab)?.accent ?? { rgb: '205, 239, 60', ink: '#244133' }

  const handleTabChange = (nextTab) => {
    if (nextTab === activeTab) {
      return
    }
    setActiveTab(nextTab)
  }

  return (
    <div className="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(255,255,255,1),_rgba(252,252,251,0.98)_48%,_rgba(247,247,245,0.96)_100%)]">
      <div className="mx-auto flex min-h-screen w-full max-w-[1680px] flex-col px-4 pb-32 pt-8 sm:px-6 lg:px-10">
        <div className="content-shell relative min-h-[calc(100vh-10rem)] overflow-visible">
          <div className="relative h-full px-4 py-6 sm:px-6 sm:py-8 lg:px-10 lg:py-10">
            <div className="panel-stack min-h-[calc(100vh-14rem)]">
              {outgoingComponent && (
                <div
                  className={`panel-layer ${
                    transitionDirection === 'forward' ? 'panel-exit-left' : 'panel-exit-right'
                  }`}
                >
                  {outgoingComponent}
                </div>
              )}
              <div
                className={`panel-layer ${isTransitioning ? (transitionDirection === 'forward' ? 'panel-enter-right' : 'panel-enter-left') : 'panel-static'}`}
              >
                {displayedComponent}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="pointer-events-none fixed inset-x-0 bottom-5 z-30 px-4 sm:px-6">
        <div className="mx-auto flex w-full max-w-fit justify-center">
          <div className="nav-shell pointer-events-auto">
            <div
              className="nav-group"
              style={{
                '--nav-accent-rgb': activeAccent.rgb,
                '--nav-accent-ink': activeAccent.ink,
                '--nav-accent-secondary': activeAccent.secondary ?? activeAccent.ink,
              }}
            >
              {tabs.map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  onClick={() => handleTabChange(tab.id)}
                  className={`nav-item ${activeTab === tab.id ? 'nav-item-active' : ''}`}
                >
                  {tab.icon && <span className="nav-item-icon-wrap">{tab.icon}</span>}
                  <span className="nav-item-copy">
                    <span className="nav-item-primary">{tab.primaryLabel}</span>
                    <span className="nav-item-secondary">{tab.secondaryLabel}</span>
                  </span>
                </button>
              ))}
              <a
                href="https://dash.iaminyourpc.xyz"
                target="_blank"
                rel="noreferrer"
                className="nav-item nav-linkout"
              >
                <span className="nav-linkout-icon" aria-hidden="true">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M10 13a5 5 0 0 0 7.07 0l1.41-1.41a5 5 0 0 0-7.07-7.07L10 5" />
                    <path d="M14 11a5 5 0 0 0-7.07 0L5.52 12.41a5 5 0 0 0 7.07 7.07L14 19" />
                  </svg>
                </span>
                <span>glance</span>
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
