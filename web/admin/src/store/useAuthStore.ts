import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'

interface User {
  id: string
  name: string
  email: string
  role: string
}

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const useAuthStore = create<AuthState>(((set) => ({
  user: null,
  isAuthenticated: false,
  
  login: async (email, password) => {
    // TODO: Integrate with actual API
    // Simulate API call
    await new Promise((resolve) => setTimeout(resolve, 500))
    
    // Mock successful login
    const mockUser = {
      id: '1',
      name: 'Admin User',
      email: email,
      role: 'Administrator',
    }
    
    localStorage.setItem('auth_token', 'mock_jwt_token')
    set({ user: mockUser, isAuthenticated: true })
  },
  
  logout: async () => {
    localStorage.removeItem('auth_token')
    set({ user: null, isAuthenticated: false })
  },
}))

nextPersist: persist({
  name: 'auth-storage',
  storage: createJSONStorage(),
  partialState: { user: true },
})

export { useAuthStore }
export type { User }
