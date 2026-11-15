import { useCallback, useEffect, useState } from 'react'
import { ApiError } from '../api/client'

// Minimal data-fetching hook: run on mount + a reload() to refetch after mutations.
export function useAsync<T>(fn: () => Promise<T>, deps: unknown[] = []) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const run = useCallback(() => {
    setLoading(true)
    setError(null)
    fn()
      .then((d) => setData(d))
      .catch((e) => {
        // 401 is handled globally (redirect to login); don't flash an error for it.
        if (!(e instanceof ApiError && e.status === 401)) {
          setError(e instanceof Error ? e.message : String(e))
        }
      })
      .finally(() => setLoading(false))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  useEffect(run, [run])
  return { data, error, loading, reload: run }
}
