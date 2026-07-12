import axios from 'axios'
import { useAuth } from '../store/auth'

export const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:3000/api'
})

api.interceptors.request.use(c => {
  const t = useAuth.getState().accessToken
  if (t) c.headers.Authorization = `Bearer ${t}`
  return c
})

let refreshing: Promise<string> | null = null

api.interceptors.response.use(r => r, async e => {
  const c = e.config
  if (e.response?.status !== 401 || c._retry) throw e
  c._retry = true
  refreshing ??= api.post('/auth/refresh', { refreshToken: useAuth.getState().refreshToken })
    .then(r => {
      useAuth.getState().setTokens(r.data)
      return r.data.accessToken
    })
    .finally(() => refreshing = null)
  c.headers.Authorization = `Bearer ${await refreshing}`
  return api(c)
})
