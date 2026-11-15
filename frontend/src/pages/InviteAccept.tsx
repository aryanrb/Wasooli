import { useEffect, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { api, googleLoginUrl } from '../api/client'
import type { InvitationView } from '../api/types'
import { useAuth } from '../auth'
import { Button, Card, ErrorText, Spinner } from '../ui'

export function InviteAccept() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const { user, loading } = useAuth()
  const [result, setResult] = useState<InvitationView | null>(null)
  const [error, setError] = useState<string | null>(null)
  const done = useRef(false)

  useEffect(() => {
    if (loading || !token || done.current) return
    if (!user) {
      // Come back here after Google login completes (it redirects to app origin).
      sessionStorage.setItem('postLoginRedirect', `/invite/accept?token=${token}`)
      window.location.href = googleLoginUrl()
      return
    }
    done.current = true
    api
      .post<InvitationView>('/invitations/accept', { token })
      .then(setResult)
      .catch((e) => setError(e instanceof Error ? e.message : String(e)))
  }, [loading, user, token])

  return (
    <div className="min-h-dvh flex flex-col items-center justify-center px-6">
      <div className="max-w-md w-full">
        {!token ? (
          <ErrorText>Missing invitation token.</ErrorText>
        ) : loading || (!user && !error) ? (
          <Spinner />
        ) : (
          <Card className="text-center space-y-3">
            {result ? (
              <>
                <p className="text-lg font-semibold">Invitation accepted</p>
                <p className="text-sm text-stone-500 capitalize">
                  You are now a {result.role}.
                </p>
              </>
            ) : (
              <ErrorText>{error ?? 'Could not accept invitation.'}</ErrorText>
            )}
            <Link to="/">
              <Button className="w-full">Go to dashboard</Button>
            </Link>
          </Card>
        )}
      </div>
    </div>
  )
}
