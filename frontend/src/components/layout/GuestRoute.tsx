import { Navigate, Outlet } from 'react-router-dom'
import { getAccessToken } from '../../api/authStorage'

export function GuestRoute() {
  if (getAccessToken()) {
    return <Navigate to="/" replace />
  }
  return <Outlet />
}
