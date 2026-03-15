import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate, Link } from 'react-router-dom'
import Login from './pages/Login'
import Settings from './pages/Settings'
import { checkAuth } from './api/client'

function NavBar() {
  return (
    <header className="sticky top-0 z-10 bg-white border-b border-gray-200">
      <div className="max-w-5xl mx-auto px-4 h-14 flex items-center justify-between">
        <Link
          to="/"
          className="text-lg font-semibold text-gray-900 hover:text-blue-600 transition-colors"
        >
          Finance Visualizer
        </Link>
        <nav>
          <Link
            to="/settings"
            className="text-sm font-medium text-gray-600 hover:text-blue-600 transition-colors"
          >
            Settings
          </Link>
        </nav>
      </div>
    </header>
  )
}

function Dashboard({ onNavigateSettings }: { onNavigateSettings: () => void }) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="text-center">
        <h1 className="text-3xl font-semibold text-gray-900 mb-2">
          Finance Visualizer
        </h1>
        <p className="text-gray-600 mb-4">Dashboard coming soon.</p>
        <button
          type="button"
          onClick={onNavigateSettings}
          className="text-sm text-blue-600 hover:text-blue-800 font-medium"
        >
          Go to Settings &rarr;
        </button>
      </div>
    </div>
  )
}

function AuthenticatedApp() {
  return (
    <BrowserRouter>
      <NavBar />
      <Routes>
        <Route
          path="/"
          element={
            <Dashboard
              onNavigateSettings={() => {
                window.location.href = '/settings'
              }}
            />
          }
        />
        <Route
          path="/settings"
          element={
            <Settings
              onNavigateDashboard={() => {
                window.location.href = '/'
              }}
            />
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}

function App() {
  const [authenticated, setAuthenticated] = useState<boolean | null>(null)

  useEffect(() => {
    checkAuth().then(setAuthenticated)
  }, [])

  if (authenticated === null) {
    return null
  }

  if (!authenticated) {
    return <Login onSuccess={() => setAuthenticated(true)} />
  }

  return <AuthenticatedApp />
}

export default App
