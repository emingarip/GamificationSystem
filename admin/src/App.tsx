import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Suspense, lazy, useState } from 'react'
import { Toaster } from 'sonner'
import { AuthProvider, useAuth } from '@/context/AuthContext'
import Sidebar from '@/components/Sidebar'
import ThemeToggle from '@/components/ThemeToggle'

const Login = lazy(() => import('@/pages/Login'))
const Dashboard = lazy(() => import('@/pages/Dashboard'))
const Rules = lazy(() => import('@/pages/Rules'))
const EventTypes = lazy(() => import('@/pages/EventTypes'))
const Users = lazy(() => import('@/pages/Users'))
const Badges = lazy(() => import('@/pages/Badges'))
const Analytics = lazy(() => import('@/pages/Analytics'))
const EventDebugger = lazy(() => import('@/pages/EventDebugger'))
const Settings = lazy(() => import('@/pages/Settings'))
const UserDetail = lazy(() => import('@/pages/UserDetail'))

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      retry: 1,
    },
  },
})

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

function RouteLoader() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
    </div>
  )
}

function AppLayout({ children }: { children: React.ReactNode }) {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)

  return (
    <div className="flex h-screen bg-background">
      <Sidebar collapsed={sidebarCollapsed} onToggle={() => setSidebarCollapsed(!sidebarCollapsed)} />
      <div className="flex-1 flex flex-col overflow-hidden">
        <header className="h-16 flex items-center justify-between px-6 border-b border-border bg-card">
          <h1 className="text-lg font-semibold text-foreground">Admin Dashboard</h1>
          <div className="flex items-center gap-4">
            <ThemeToggle />
          </div>
        </header>
        <main className="flex-1 overflow-y-auto p-6">{children}</main>
      </div>
    </div>
  )
}

function PageBoundary({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<RouteLoader />}>{children}</Suspense>
}

function AppRoutes() {
  return (
    <Routes>
      <Route
        path="/login"
        element={
          <PageBoundary>
            <Login />
          </PageBoundary>
        }
      />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <AppLayout>
              <PageBoundary>
                <Dashboard />
              </PageBoundary>
            </AppLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/rules"
        element={
          <ProtectedRoute>
            <AppLayout>
              <PageBoundary>
                <Rules />
              </PageBoundary>
            </AppLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/event-types"
        element={
          <ProtectedRoute>
            <AppLayout>
              <PageBoundary>
                <EventTypes />
              </PageBoundary>
            </AppLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/users"
        element={
          <ProtectedRoute>
            <AppLayout>
              <PageBoundary>
                <Users />
              </PageBoundary>
            </AppLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/users/:id"
        element={
          <ProtectedRoute>
            <AppLayout>
              <PageBoundary>
                <UserDetail />
              </PageBoundary>
            </AppLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/badges"
        element={
          <ProtectedRoute>
            <AppLayout>
              <PageBoundary>
                <Badges />
              </PageBoundary>
            </AppLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/analytics"
        element={
          <ProtectedRoute>
            <AppLayout>
              <PageBoundary>
                <Analytics />
              </PageBoundary>
            </AppLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/debugger"
        element={
          <ProtectedRoute>
            <AppLayout>
              <PageBoundary>
                <EventDebugger />
              </PageBoundary>
            </AppLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/settings"
        element={
          <ProtectedRoute>
            <AppLayout>
              <PageBoundary>
                <Settings />
              </PageBoundary>
            </AppLayout>
          </ProtectedRoute>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
        <Toaster position="top-right" richColors />
      </AuthProvider>
    </QueryClientProvider>
  )
}
