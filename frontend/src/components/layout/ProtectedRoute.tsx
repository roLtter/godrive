import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { getAccessToken } from '../../api/authStorage'
import { CurrentUserProvider } from '../../contexts/CurrentUserContext'

export function ProtectedRoute() {
  const location = useLocation()

  if (!getAccessToken()) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return (
    <CurrentUserProvider>
      <Outlet />
    </CurrentUserProvider>
  )
}
