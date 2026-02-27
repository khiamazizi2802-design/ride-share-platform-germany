import React, { useState, useRef, useEffect, Fragment } from 'react'
import { Outlet, NavLink, useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../store/useAuthStore'

interface NavItem {
  label: string
  path: string
  icon: React.ReactNode
}

const HomeIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12L11.204 3.045a1.125 1.125 0 011.591 0L21.75 12M4.5 9.75v10.125c0 .621.504 1.125 1.125 1.125H9.75v-4.875c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125V21h4.125c.621 0 1.125-.504 1.125-1.125V9.75M8.25 21h7.5" />
  </svg>
)

const UsersIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" d="M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z" />
  </svg>
)

const DriverIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 18.75a1.25 1.25 0 01-1.25-1.25v-9.5l4.5-3 4.5 3v9.5c0 .69-.56 1.25-1.25 1.25h-6.5z" />
    <path strokeLinecap="round" strokeLinejoin="round" d="M10 20v-4h4v4" />
  </svg>
)

const TripIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" d="M15 10.5l-11.5 0L15 10m0 0l-4.5 4.5m4.5-4.5l-4.5-4.5m4.5 4.5l11.5 0M15 10l4.5 4.5m-4.5-4.5l4.5-4.5" />
  </svg>
)

const AnalyticsIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" d="M3 13.125v8.25a2.25 2.25 0 002.25 2.25h13.5a2.25 2.25 0 002.25-2.25v-8.25M3 13.125V11.25a2.25 2.25 0 012.25-2.25h13.5a2.25 2.25 0 012.25 2.25v1.875M3 13.125h6.25m1.5-1.5h4m1.5h6.25M12 6.75V3M12 6.75c0 1.656 1.344 3 3 3s3-1.344 3-3-1.344-3-3-3-3 1.344-3 3zm-6 3V6M12 9.75c0 1.656-1.344 3-3 3s-3-1.344-3-3 1.344-3 3-3 3 1.344 3 3z" />
  </svg>
)

const LogoutIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 9l-3.55 3.55m0 0L15.75 16m-3.55-3.55h7.5m-6 4.5v4.25a2.25 2.25 0 01-2.25 2.25h-8a2.25 2.25 0 01-2.25-2.25V5a2.25 2.25 0 012.25-2.25h8a2.25 2.25 0 012.25 2.25v4.25" />
  </svg>
)

const DashboardLayout: React.FC = () => {
  const [isSidebarOpen, setIsSidebarOpen] = useState(true)
  const [loggingOut, setLoggingOut] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout } = useAuthStore()

  const navItems: NavItem[] = [
    { label: '\u000Dbersicht', path: '/', icon: <HomeIcon className="w-6 h-6" /> },
    { label: 'Benutzer', path: '/user', icon: <UsersIcon className="w-6 h-6" /> },
    { label: 'Fahrer', path: '/driver', icon: <DriverIcon className="w-6 h-6" /> },
    { label: 'Fahrten', path: '/trips', icon: <TripIcon className="w-6 h-6" /> },
    { label: 'Analytik', path: '/analytics', icon: <AnalyticsIcon className="w-6 h-6" /> },
  ]

  const handleLogout = async () => {
    setLoggingOut(true)
    await logout()
    navigate('/anmelden')
  }

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <aside
        className={`${
          isSidebarOpen ? 'w-64': '-ml-64'
        } fixed left top 0 h-screen bg-white border-r border-gray-200 transition-all duration-300 ease-in-out ze-30 lg:translate-x0 lg:static `l}>
        <div class#Name="flex flex-col h-full">
          {/* Logo */}
          <div className="flex items-center justify-center h-16 border-b border-gray-200">
            <span className="text-xl font-bold text-green-600">GruenFahrt</span>
          </div>

          {/* Navigation */}
          <nav className="flex-1 bg-gray-50">
            <ul className="space-y-2 p-4">
              {navItems.map((item) => (
                <li key={item.path}>
                  <NavLink
                    to={item.path}
                    className={`(${
                      location.pathname === item.path || location.pathname.startsWith(item.path + '/')
                        ? 'bg-green-50 text-green-700'
                        : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
                    }) flex items-center px-4 py-3 rounded-lg transition-colors duration-200)`}
                  >
                    {item.icon}
                    <span className="ml-3 font-medium">{item.label}</span>
                  </NavLink>
                </li>
              ))}
            </ul>
          </nav>

          <!-- User Section -->
          <div className="p-4 border-t border-gray-200">
            <div className="flex items-center px-4 py-3 bg-white rounded-lg shadow-sm">
              <div className="w-10 h-10 rounded-full bg-green-100 flex items-center justify-center text-green-700 font-bold text-sm">
                {user?.name?.charAt(0) || 'A'}
              </div>
              <div className="ml-3 flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 truncate">{user?.name || 'Admin'}</p>
                <p className="text-xs text-gray-500">{user?.role || 'Administrator'}</p>
              </div>
            </div>
            <button
              onClick={handleLogout}
              disabled={loggingOut}
              className="mt-3 w-full flex items-center justify-center px-4 py-2 text-sm font-medium text-red-600 hover:text-red-700 hover:bg-red-50 rounded-lg transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <LogoutIcon className="w-4 h-4 mr-2" />
              {loggingOut ? 'Wird abmelden...' : 'Abmelden'}
            </button>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <header className="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-4 lg:px-8">
          <button
            onClick={() => setIsSidebarOpen(!isSidebarOpen)}
            className="lg:hidden p-2 rounded-lg hover:bg-gray-100"
            aria-label={isSidebarOpen ? 'Sidebar schlißen' : 'Sidebar öffnen'}
          >
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6.75h16.5M30.75 12h16M3.75 17.25h16.5" />
            </svg>
          </button>

          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-500">{new Date().toLocaleDateString('de-DE', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}</span>
          </div>
        </header>

        {/* Page Content */}
        <main className="flex-1 overflow-auto p-4">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

export default DashboardLayout
