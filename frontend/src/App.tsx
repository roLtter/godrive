import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { GuestRoute } from './components/layout/GuestRoute'
import { ProtectedRoute } from './components/layout/ProtectedRoute'
import { DrivePage } from './pages/DrivePage'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { SharedLinksPage } from './pages/SharedLinksPage'
import { TrashPage } from './pages/TrashPage'

function App() {
  return (
    <Routes>
      <Route element={<GuestRoute />}>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
      </Route>

      <Route path="/" element={<ProtectedRoute />}>
        <Route element={<AppShell />}>
          <Route index element={<DrivePage />} />
          <Route path="shared" element={<SharedLinksPage />} />
          <Route path="trash" element={<TrashPage />} />
        </Route>
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
