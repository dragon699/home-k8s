import React, { useState } from 'react'
import ActionCenter from './components/ActionCenter'
import './App.css'

function App() {
  const [activeTab, setActiveTab] = useState('fetch-api')

  return (
    <div className="min-h-screen bg-gray-50">
      <ActionCenter activeTab={activeTab} setActiveTab={setActiveTab} />
    </div>
  )
}

export default App
