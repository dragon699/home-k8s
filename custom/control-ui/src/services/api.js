const API_BASE_URL = import.meta.env.VITE_API_URL || ''

export async function addTorrent(payload) {
  try {
    const response = await fetch(`${API_BASE_URL}/torrents/`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    })

    if (!response.ok) {
      const errorBody = await response.json().catch(() => ({}))
      throw new Error(errorBody.error || `HTTP error! status: ${response.status}`)
    }

    return await response.json()
  } catch (error) {
    console.error('Error adding torrent:', error)
    throw error
  }
}

export async function getTorrents() {
  try {
    const response = await fetch(`${API_BASE_URL}/torrents/`)
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    return await response.json()
  } catch (error) {
    console.error('Error fetching torrents:', error)
    throw error
  }
}

export async function getHealth() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/health`)
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    return await response.json()
  } catch (error) {
    console.error('Error fetching health:', error)
    throw error
  }
}

export async function getReady() {
  try {
    const response = await fetch(`${API_BASE_URL}/api/ready`)
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    return await response.json()
  } catch (error) {
    console.error('Error fetching ready status:', error)
    throw error
  }
}
