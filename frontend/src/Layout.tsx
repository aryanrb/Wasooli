import type { ReactNode } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from './auth'

function TabLink({ to, label, active }: { to: string; label: string; active: boolean }) {
  return (
    <Link
      to={to}
      className={`flex-1 py-3 text-center text-xs font-medium ${
        active ? 'text-stone-900' : 'text-stone-400'
      }`}
    >
      {label}
    </Link>
  )
}

export function Layout({ title, children }: { title?: string; children: ReactNode }) {
  const { pathname } = useLocation()
  const { user, logout } = useAuth()

  return (
    <div className="min-h-dvh">
      <header className="sticky top-0 z-40 border-b border-stone-200 bg-white/90 backdrop-blur">
        <div className="max-w-md mx-auto flex items-center justify-between px-4 h-14">
          <h1 className="font-semibold truncate">{title ?? 'Wasooli'}</h1>
          {user && (
            <button onClick={() => void logout()} className="text-xs text-stone-500 hover:text-stone-900">
              Sign out
            </button>
          )}
        </div>
      </header>

      <main className="max-w-md mx-auto px-4 py-4 pb-24 space-y-4">{children}</main>

      <nav className="fixed bottom-0 inset-x-0 z-40 border-t border-stone-200 bg-white">
        <div className="max-w-md mx-auto flex">
          <TabLink to="/" label="Home" active={pathname === '/'} />
          <TabLink to="/approvals" label="Approvals" active={pathname === '/approvals'} />
          <TabLink to="/analytics" label="Analytics" active={pathname === '/analytics'} />
        </div>
      </nav>
    </div>
  )
}
