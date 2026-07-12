import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type S = {
  accessToken: string | null
  refreshToken: string | null
  setTokens: (x: { accessToken: string; refreshToken: string }) => void
  clear: () => void
}

export const useAuth = create<S>()(
  persist(
    set => ({
      accessToken: null,
      refreshToken: null,
      setTokens: x => {
        set(x)
        // Also set cookies for SSR
        if (typeof document !== 'undefined') {
          document.cookie = `accessToken=${x.accessToken}; path=/; max-age=3600`
          document.cookie = `refreshToken=${x.refreshToken}; path=/; max-age=2592000`
        }
      },
      clear: () => {
        set({ accessToken: null, refreshToken: null })
        if (typeof document !== 'undefined') {
          document.cookie = 'accessToken=; path=/; max-age=0'
          document.cookie = 'refreshToken=; path=/; max-age=0'
        }
      }
    }),
    { name: 'cashier-auth' }
  )
)
