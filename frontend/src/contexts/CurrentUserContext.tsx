import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from 'react'
import { useNavigate } from 'react-router-dom'
import { fetchMe, type MeResponse } from '../api/me'
import { getAccessToken } from '../api/authStorage'
import { toApiError } from '../api/client'

type Status = 'loading' | 'ready' | 'error'

type CurrentUserContextValue = {
  user: MeResponse | null
  status: Status
  errorMessage: string | null
  refresh: () => Promise<void>
}

const CurrentUserContext = createContext<CurrentUserContextValue | null>(null)

export function CurrentUserProvider({ children }: PropsWithChildren) {
  const navigate = useNavigate()
  const [user, setUser] = useState<MeResponse | null>(null)
  const [status, setStatus] = useState<Status>('loading')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    if (!getAccessToken()) {
      setUser(null)
      setStatus('error')
      setErrorMessage('Not signed in')
      navigate('/login', { replace: true })
      return
    }
    setStatus('loading')
    setErrorMessage(null)
    try {
      const me = await fetchMe()
      setUser(me)
      setStatus('ready')
    } catch (err) {
      setUser(null)
      setStatus('error')
      setErrorMessage(toApiError(err).message)
      navigate('/login', { replace: true })
    }
  }, [navigate])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const value = useMemo(
    () => ({ user, status, errorMessage, refresh }),
    [user, status, errorMessage, refresh],
  )

  return <CurrentUserContext.Provider value={value}>{children}</CurrentUserContext.Provider>
}

export function useCurrentUser() {
  const ctx = useContext(CurrentUserContext)
  if (!ctx) {
    throw new Error('useCurrentUser must be used within CurrentUserProvider')
  }
  return ctx
}
