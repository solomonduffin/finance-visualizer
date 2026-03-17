import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route, Navigate, Link } from 'react-router-dom'
import Login from './pages/Login'
import Settings from './pages/Settings'
import Dashboard from './pages/Dashboard'
import NetWorth from './pages/NetWorth'
import Alerts from './pages/Alerts'
import Projections from './pages/Projections'
import { checkAuth } from './api/client'
import { useDarkMode } from './hooks/useDarkMode'

// Sun icon SVG (for when dark mode is active — click to switch to light)
function SunIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      width="18"
      height="18"
      aria-hidden="true"
    >
      <circle cx="12" cy="12" r="5" />
      <line x1="12" y1="1" x2="12" y2="3" />
      <line x1="12" y1="21" x2="12" y2="23" />
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
      <line x1="1" y1="12" x2="3" y2="12" />
      <line x1="21" y1="12" x2="23" y2="12" />
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
    </svg>
  )
}

// Moon icon SVG (for when dark mode is inactive — click to switch to dark)
function MoonIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      width="18"
      height="18"
      aria-hidden="true"
    >
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  )
}

function NavBar({ isDark, onToggle }: { isDark: boolean; onToggle: () => void }) {
  return (
    <header className="sticky top-0 z-10 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
      <div className="max-w-5xl mx-auto px-4 h-14 flex items-center justify-between">
        <Link
          to="/"
          className="text-lg font-semibold text-gray-900 dark:text-gray-100 hover:text-blue-600 transition-colors"
        >
          Finance Visualizer
        </Link>
        <nav className="flex items-center gap-3">
          <button
            type="button"
            onClick={onToggle}
            aria-label={isDark ? 'Enable light mode' : 'Enable dark mode'}
            className="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors text-gray-600 dark:text-gray-400"
          >
            {isDark ? <SunIcon /> : <MoonIcon />}
          </button>
          <Link
            to="/net-worth"
            className="text-sm font-semibold text-gray-600 dark:text-gray-400 hover:text-blue-600 transition-colors"
          >
            Net Worth
          </Link>
          <Link
            to="/alerts"
            className="text-sm font-semibold text-gray-600 dark:text-gray-400 hover:text-blue-600 transition-colors"
          >
            Alerts
          </Link>
          <Link
            to="/projections"
            className="text-sm font-semibold text-gray-600 dark:text-gray-400 hover:text-blue-600 transition-colors"
          >
            Projections
          </Link>
          <Link
            to="/settings"
            className="text-sm font-semibold text-gray-600 dark:text-gray-400 hover:text-blue-600 transition-colors"
          >
            Settings
          </Link>
        </nav>
      </div>
    </header>
  )
}

function AuthenticatedApp() {
  const { isDark, toggle } = useDarkMode()

  return (
    <BrowserRouter>
      <NavBar isDark={isDark} onToggle={toggle} />
      <Routes>
        <Route path="/" element={<Dashboard />} />
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
        <Route path="/net-worth" element={<NetWorth />} />
        <Route path="/alerts" element={<Alerts />} />
        <Route path="/projections" element={<Projections />} />
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
