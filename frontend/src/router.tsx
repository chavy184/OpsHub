import { BrowserRouter, Routes, Route, Link, Navigate, useLocation } from 'react-router-dom'
import AppLayout from '@/components/layout/app-layout'
import { isAuthenticated } from '@/lib/auth'
import ServicesPage from '@/pages/services'
import ServiceDetailPage from '@/pages/service-detail'
import ReleasesPage from '@/pages/releases'
import ReleaseDetailPage from '@/pages/release-detail'
import CredentialsPage from '@/pages/credentials'
import HostsPage from '@/pages/hosts'
import ContainersPage from '@/pages/containers'
import ImageSyncPage from '@/pages/image-sync'
import BackupsPage from '@/pages/backups'
import DocumentsPage from '@/pages/documents'
import ArchivesPage from '@/pages/archives'
import LogsPage from '@/pages/logs'
import SettingsPage from '@/pages/settings'
import MonitorPage from '@/pages/monitor'
import AlertsPage from '@/pages/alerts'
import NotificationsPage from '@/pages/notifications'
import LoginPage from '@/pages/login'

function NotFoundPage() {
  return (
    <div className="flex flex-col items-center justify-center gap-4 py-32">
      <h1 className="text-6xl font-bold text-muted-foreground">404</h1>
      <p className="text-muted-foreground">页面不存在</p>
      <Link to="/services" className="text-sm text-primary hover:underline">返回首页</Link>
    </div>
  )
}

function RequireAuth() {
  const location = useLocation()
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace state={{ from: location }} />
  }
  return <AppLayout />
}

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<RequireAuth />}>
          <Route path="/" element={<Navigate to="/services" replace />} />
          <Route path="/services" element={<ServicesPage />} />
          <Route path="/services/:id" element={<ServiceDetailPage />} />
          <Route path="/releases" element={<ReleasesPage />} />
          <Route path="/releases/:id" element={<ReleaseDetailPage />} />
          <Route path="/credentials" element={<CredentialsPage />} />
          <Route path="/hosts" element={<HostsPage />} />
          <Route path="/hosts/image-sync" element={<ImageSyncPage />} />
          <Route path="/containers" element={<ContainersPage />} />
          <Route path="/backups" element={<Navigate to="/backups/tasks" replace />} />
          <Route path="/backups/tasks" element={<BackupsPage />} />
          <Route path="/backups/records" element={<BackupsPage />} />
          <Route path="/backups/migrations" element={<BackupsPage />} />
          <Route path="/backups/migration-records" element={<BackupsPage />} />
          <Route path="/backups/object-sync" element={<BackupsPage />} />
          <Route path="/backups/object-sync-records" element={<BackupsPage />} />
          <Route path="/documents" element={<DocumentsPage />} />
          <Route path="/archives" element={<ArchivesPage />} />
          <Route path="/monitor" element={<MonitorPage />} />
          <Route path="/alerts" element={<AlertsPage />} />
          <Route path="/notifications" element={<NotificationsPage />} />
          <Route path="/logs" element={<LogsPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
