import React, { Suspense, lazy } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import DashboardLayout from './layouts/DashboardLayout'
import { useAuthStore } from './store/useAuthStore'

const DashboardPage = lazy(() => import('./pages/DashboardPage'))
const UsersPage = lazy(() => import('./pages/UsersPage'))
const DriversPage = lazy(() => import('./pages/DriversPage'))
const TripsPage = lazy(() => import('./pages/TripsPage'))
const AnalyticsPage = lazy(() => import('./pages/AnalyticsPage'))
const LoginPage = lazy(() => import('./pages/LoginPage'))
const NotFoundPage = lazy(() => import('./pages/NotFoundPage'))

interface ProtectedRouteProps {
  children: React.ReactNode
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children }) => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  if (!isAuthenticated) {
    return <Navigate to="/anmelden" replace />
  }
  return <>{children}</>
}

const PublicOnlyRoute: React.FC<ProtectedRouteProps> = ({ children }) => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}

const App: React.FC = () => {
  return (
    <BrowserRouter>
      <Suspense fallback={<PageLoader />}>
        <Routes>
          <Route path="/anmelden" element={
            <PublicOnlyRoute>
              <LoginPage />
            </PublicOnlyRoute>
          } />
          <Route path="/" element={(
            <ProtectedRoute>
              <DashboardLayout>
                <Routes>
                  <Route index element={<DashboardPage />} />
                  <Route path="user" element={<UsersPage />} />
                  <Route path="driver" element={<DriversPage />} />
                  <Route path="trips" element={<TripsPage />} />
                  <Route path="analytics" element={<AnalyticsPage />} />
                </Routes>
              </DashboardLayout>
            </ProtectedRoute>
          )} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </Suspense>
    </BrowserRouter>
  )
}

const PageLoader: React.FC = () => (
  <div className="flex items-center justify-center min-h-screen bg-gray-50">
    <div className="flex flex-col items-center gap-4">
      <div className="w-12 h-12 border-4 border-blue-600 border-t-transparent rounded-full animate-spin" />
      <p className="text-sm text-gray-500 font-medium">Wird geladen...</p>
    </div>
  </div>
)

export default App
