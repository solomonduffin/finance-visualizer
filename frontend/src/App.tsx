import { useState, useEffect } from 'react'
import Login from './pages/Login'
import { checkAuth } from './api/client'

function App() {
  const [authenticated, setAuthenticated] = useState<boolean | null>(null)

  useEffect(() => {
    checkAuth().then(setAuthenticated)
  }, [])

  if (authenticated === null) {
    // Still checking auth — show nothing (or a spinner)
    return null
  }

  if (!authenticated) {
    return <Login onSuccess={() => setAuthenticated(true)} />
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="text-center">
        <h1 className="text-3xl font-semibold text-gray-900 mb-2">
          Finance Visualizer
        </h1>
        <p className="text-gray-600">Dashboard coming soon.</p>
      </div>
    </div>
  )
}

export default App
