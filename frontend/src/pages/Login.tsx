import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { googleLoginUrl } from '../api/client'
import { useAuth } from '../auth'
import { Button, Spinner } from '../ui'

export function Login() {
  const { user, loading } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (!loading && user) {
      const back = sessionStorage.getItem('postLoginRedirect')
      sessionStorage.removeItem('postLoginRedirect')
      navigate(back || '/', { replace: true })
    }
  }, [loading, user, navigate])

  if (loading) return <Spinner />

  return (
    <div className="min-h-dvh flex flex-col items-center justify-center px-6 text-center">
      <div className="max-w-md w-full space-y-6">
        <div>
          <h1 className="text-3xl font-semibold">Wasooli</h1>
          <p className="text-stone-500 mt-2">Rental-income ledger</p>
        </div>
        <Button className="w-full" onClick={() => (window.location.href = googleLoginUrl())}>
          Continue with Google
        </Button>
      </div>
    </div>
  )
}
