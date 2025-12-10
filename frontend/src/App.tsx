import './App.css'
import './index.css'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { AnalyzePage } from './pages/AnalyzePage'
import { PositionPage } from './pages/PositionPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<AnalyzePage />} />
        <Route path="/position/:id" element={<PositionPage />} />
      </Routes>
    </BrowserRouter>
  )
}
