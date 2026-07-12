'use client'

import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { api } from '@/lib/api'
import { useAuth } from '@/store/auth'

export default function LoginPage() {
  const f = useForm()
  const router = useRouter()
  
  const onSubmit = async (data: any) => {
    const r = await api.post('/auth/login', data)
    useAuth.getState().setTokens(r.data)
    router.push('/dashboard')
  }

  return (
    <main className="min-h-screen grid place-items-center">
      <form className="card w-96 space-y-4" onSubmit={f.handleSubmit(onSubmit)}>
        <h1>Вход в систему</h1>
        <input className="w-full" placeholder="Email" {...f.register('email')} />
        <input className="w-full" type="password" placeholder="Пароль" {...f.register('password')} />
        <button className="w-full">Войти</button>
      </form>
    </main>
  )
}
