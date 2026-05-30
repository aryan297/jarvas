import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import Layout from '@/components/layout/Layout'
import LoginPage from '@/pages/auth/LoginPage'
import DashboardPage from '@/pages/dashboard/DashboardPage'
import ChatPage from '@/pages/chat/ChatPage'
import DocumentsPage from '@/pages/documents/DocumentsPage'
import AgentsPage from '@/pages/agents/AgentsPage'
import MemoryPage from '@/pages/memory/MemoryPage'
import WorkflowsPage from '@/pages/workflows/WorkflowsPage'
import SettingsPage from '@/pages/settings/SettingsPage'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />
}

function PublicRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return !isAuthenticated ? <>{children}</> : <Navigate to="/dashboard" replace />
}

export default function App() {
  return (
    <Routes>
      {/* Public routes — /register redirects to /login which has a built-in Register tab */}
      <Route path="/login"    element={<PublicRoute><LoginPage /></PublicRoute>} />
      <Route path="/register" element={<Navigate to="/login" replace />} />

      {/* Protected routes */}
      <Route path="/" element={<PrivateRoute><Layout /></PrivateRoute>}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard"  element={<DashboardPage />} />
        <Route path="chat"       element={<ChatPage />} />
        <Route path="chat/:id"   element={<ChatPage />} />
        <Route path="documents"  element={<DocumentsPage />} />
        <Route path="agents"     element={<AgentsPage />} />
        <Route path="memory"     element={<MemoryPage />} />
        <Route path="workflows"  element={<WorkflowsPage />} />
        <Route path="settings"   element={<SettingsPage />} />
      </Route>

      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  )
}
