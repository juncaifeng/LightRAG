import { useEffect } from 'react'
import { HashRouter, Routes, Route } from 'react-router-dom'
import { Toaster } from 'sonner'
import { ThemeProvider } from '@/components/ThemeProvider'
import { ThemeToggle } from '@/components/ThemeToggle'
import { LangToggle } from '@/components/LangToggle'
import { I18nProvider } from '@/lib/i18n'
import { Navigation } from '@/components/Navigation'
import { ConnectionStatus } from '@/components/ConnectionStatus'
import { Separator } from '@/components/ui/separator'
import { useDashboardStore } from '@/stores/dashboard'
import { OverviewPage } from '@/pages/OverviewPage'
import { SubscribersPage } from '@/pages/SubscribersPage'
import { EventStreamPage } from '@/pages/EventStreamPage'
import { MetricsPage } from '@/pages/MetricsPage'
import { TopicsPage } from '@/pages/TopicsPage'
import { ServicesPage } from '@/pages/ServicesPage'

function AppLayout() {
  const startPolling = useDashboardStore((s) => s.startPolling)
  const stopPolling = useDashboardStore((s) => s.stopPolling)
  const connectWS = useDashboardStore((s) => s.connectWS)
  const disconnectWS = useDashboardStore((s) => s.disconnectWS)

  useEffect(() => {
    startPolling()
    connectWS()
    return () => {
      stopPolling()
      disconnectWS()
    }
  }, [startPolling, stopPolling, connectWS, disconnectWS])

  return (
    <div className="flex h-screen">
      {/* Sidebar */}
      <aside className="w-56 border-r bg-background flex flex-col">
        <div className="flex items-center justify-between px-4 h-14 border-b">
          <h1 className="font-bold text-lg">EventBus</h1>
          <ThemeToggle />
          <LangToggle />
        </div>
        <div className="flex-1 py-4">
          <Navigation />
        </div>
        <div className="px-4 py-3 border-t">
          <ConnectionStatus />
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <div className="p-6">
          <Routes>
            <Route path="/" element={<OverviewPage />} />
            <Route path="/subscribers" element={<SubscribersPage />} />
            <Route path="/events" element={<EventStreamPage />} />
            <Route path="/topics" element={<TopicsPage />} />
            <Route path="/services" element={<ServicesPage />} />
            <Route path="/metrics" element={<MetricsPage />} />
          </Routes>
        </div>
      </main>
    </div>
  )
}

export default function App() {
  return (
    <ThemeProvider defaultTheme="system">
      <I18nProvider>
        <HashRouter>
          <AppLayout />
        </HashRouter>
        <Toaster richColors position="bottom-right" />
      </I18nProvider>
    </ThemeProvider>
  )
}
