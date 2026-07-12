'use client'

import { useAuth } from '@/store/auth'
import Link from 'next/link'
import { usePathname } from 'next/navigation'

export default function Layout({ children }: { children: React.ReactNode }) {
  const token = useAuth(s => s.accessToken)
  const pathname = usePathname()

  if (!token) {
    return null // Will redirect via middleware
  }

  const links = [
    ['/dashboard', 'Панель'],
    ['/events', 'События'],
    ['/stores', 'Магазины'],
    ['/workplaces', 'Рабочие места'],
    ['/cameras', 'Камеры'],
    ['/streams', 'Потоки'],
    ['/receipts', 'Чеки'],
    ['/product-scans', 'Сканы'],
    ['/sale-sessions', 'Продажи'],
    ['/transcripts', 'Транскрипты'],
    ['/integration-errors', 'Ошибки'],
    ['/workers', 'Воркеры'],
    ['/rules', 'Правила'],
    ['/models', 'Модели'],
    ['/users', 'Пользователи']
  ]

  return (
    <div className="min-h-screen md:grid md:grid-cols-[240px_1fr]">
      <aside className="p-5 bg-slate-900 nav">
        <b className="text-cyan-400 text-xl">Cashier Copilot</b>
        <nav className="mt-8 max-h-[calc(100vh-120px)] space-y-1 overflow-y-auto pr-1">
          {links.map(x => (
            <Link 
              key={x[0]} 
              href={x[0]}
              className={pathname === x[0] ? 'active' : ''}
            >
              {x[1]}
            </Link>
          ))}
        </nav>
      </aside>
      <main className="p-6 md:p-10">
        {children}
      </main>
    </div>
  )
}
