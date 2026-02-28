import React, { useEffect, useMemo, useRef, useState } from 'react'
import ApplicationActions from './ApplicationActions'
import FetchApiActions from './FetchApiActions'

export default function ActionCenter({ activeTab, setActiveTab }) {
  const tabs = useMemo(
    () => [
      {
        id: 'fetch-api',
        label: 'fetch-api',
        icon: 'https://i.imgur.com/VwpKmbC.png',
        component: <FetchApiActions />,
      },
      { id: 'overview', label: 'overview', component: <ApplicationActions /> },
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

  const handleTabChange = (nextTab) => {
    if (nextTab === activeTab) {
      return
    }
    setActiveTab(nextTab)
  }

  return (
    <div className="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(255,255,255,0.9),_rgba(242,240,237,0.95)_48%,_rgba(229,224,216,0.85)_100%)]">
      <div className="mx-auto flex min-h-screen w-full max-w-[1680px] flex-col px-4 pb-32 pt-8 sm:px-6 lg:px-10">
        <div className="content-shell relative min-h-[calc(100vh-10rem)] overflow-hidden rounded-[2rem] border border-[#d8d1c7] bg-[#fbfaf8] shadow-[0_30px_80px_rgba(32,24,16,0.12)]">
          <div className="pointer-events-none absolute inset-x-[3%] top-0 h-px bg-[linear-gradient(90deg,rgba(197,57,42,0),rgba(197,57,42,0.35),rgba(197,57,42,0))]" />
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
            <div className="nav-group">
              {tabs.map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  onClick={() => handleTabChange(tab.id)}
                  className={`nav-item ${activeTab === tab.id ? 'nav-item-active' : ''}`}
                >
                  {tab.icon && <img src={tab.icon} alt="" className="nav-item-icon" />}
                  <span>{tab.label}</span>
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
