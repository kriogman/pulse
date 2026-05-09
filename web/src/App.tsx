import { Routes, Route } from 'react-router-dom'
import MonitorListPage from './pages/MonitorListPage'
import CheckHistoryPage from './pages/CheckHistoryPage'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<MonitorListPage />} />
      <Route path="/monitors/:id/checks" element={<CheckHistoryPage />} />
    </Routes>
  )
}
