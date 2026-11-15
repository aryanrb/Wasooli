import { useEffect, type ReactNode } from 'react'
import { Navigate, Route, BrowserRouter, Routes, useLocation, useNavigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './auth'
import { Spinner } from './ui'
import { Login } from './pages/Login'
import { Dashboard } from './pages/Dashboard'
import { Approvals } from './pages/Approvals'
import { Analytics } from './pages/Analytics'
import { PropertyPage } from './pages/Property'
import { ContractPage } from './pages/Contract'
import { InviteAccept } from './pages/InviteAccept'

function RequireAuth({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <Spinner />
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

// After OAuth returns to the app origin, send the user to any stashed invite path.
function PostLoginRedirect() {
  const { user, loading } = useAuth()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  useEffect(() => {
    if (loading || !user) return
    const back = sessionStorage.getItem('postLoginRedirect')
    if (back && back !== pathname) {
      sessionStorage.removeItem('postLoginRedirect')
      navigate(back, { replace: true })
    }
  }, [loading, user, pathname, navigate])
  return null
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <PostLoginRedirect />
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/invite/accept" element={<InviteAccept />} />
          <Route
            path="/"
            element={
              <RequireAuth>
                <Dashboard />
              </RequireAuth>
            }
          />
          <Route
            path="/approvals"
            element={
              <RequireAuth>
                <Approvals />
              </RequireAuth>
            }
          />
          <Route
            path="/analytics"
            element={
              <RequireAuth>
                <Analytics />
              </RequireAuth>
            }
          />
          <Route
            path="/properties/:id"
            element={
              <RequireAuth>
                <PropertyPage />
              </RequireAuth>
            }
          />
          <Route
            path="/contracts/:id"
            element={
              <RequireAuth>
                <ContractPage />
              </RequireAuth>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
