import React from 'react'
import FetchApiActions from './FetchApiActions'

export default function ActionCenter({ activeTab, setActiveTab }) {
  const selectedColor = '#34c97b'
  const selectedShadow = 'inset 0 1px 0 rgba(255,255,255,0.3), inset 0 -2px 6px rgba(0,0,0,0.18)'
  const activeAccentColor = activeTab === 'fetch-api' ? selectedColor : '#7c3aed'

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-[0_6px_14px_rgba(15,23,42,0.12)]">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-start">
            <span
              aria-label="Section icon"
              className="block w-8 h-8"
              style={{
                backgroundColor: activeAccentColor,
                boxShadow: selectedShadow,
                borderRadius: '0.5rem',
                WebkitMaskImage: 'url(https://i.imgur.com/8ZX8xsF.png)',
                maskImage: 'url(https://i.imgur.com/8ZX8xsF.png)',
                WebkitMaskSize: 'contain',
                maskSize: 'contain',
                WebkitMaskRepeat: 'no-repeat',
                maskRepeat: 'no-repeat',
                WebkitMaskPosition: 'center',
                maskPosition: 'center',
              }}
            />
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 pb-28">
        <div key={activeTab} className="tab-panel-enter">
          {activeTab === 'fetch-api' && <FetchApiActions />}
        </div>
      </div>

      {/* Footer */}
      <div className="fixed bottom-0 left-0 right-0 bg-white shadow-[0_-6px_14px_rgba(15,23,42,0.12)] z-20">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-end">
            <div className="flex space-x-2">
              <button
                onClick={() => setActiveTab('fetch-api')}
                className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                  activeTab === 'fetch-api'
                    ? 'text-gray-900'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                }`}
                style={activeTab === 'fetch-api' ? { backgroundColor: selectedColor, boxShadow: selectedShadow } : undefined}
              >
                <div className="flex items-center space-x-2">
                  <img src="https://i.imgur.com/URgCIOW.png" alt="" className="w-5 h-5 object-contain" />
                  <span>fetch-api</span>
                </div>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
