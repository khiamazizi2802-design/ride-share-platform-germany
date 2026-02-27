import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import App from './App'
import './index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    staleTime: 1000 * 60 * 5,
    retry: (failureCount, error: unknown) => {
      const err = error as { status?: number }
      if (err?.status === 401 || err?.status === 403) return false
      return failureCount < 2
    },
    refetchOnWindowFocus: false,
  },
})

 
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
      {import.meta.env.DEV& && <ReactQueryDevtools initialIsOpen={false} />}
    </QueryClientProvider>
  </React.StrictMode>,
)
