// Fetch wrapper: base URL + /api/v1 prefix, credentials, JSON, and error throwing.
// On 401 it dispatches a window event so the auth layer can bounce to login.

const BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080'
const API = `${BASE}/api/v1`

export const UNAUTHORIZED_EVENT = 'wasooli:unauthorized'

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    method,
    credentials: 'include',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (res.status === 401) {
    window.dispatchEvent(new CustomEvent(UNAUTHORIZED_EVENT))
    throw new ApiError(401, 'Not authenticated')
  }

  // Some endpoints (logout) may return empty bodies.
  const text = await res.text()
  const data = text ? JSON.parse(text) : null

  if (!res.ok) {
    const msg = (data && data.error) || `Request failed (${res.status})`
    throw new ApiError(res.status, msg)
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
}

// Full-page nav to the backend-hosted Google OAuth start.
export function googleLoginUrl(): string {
  return `${API}/auth/google/login`
}
