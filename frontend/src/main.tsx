import { createRoot } from 'react-dom/client'
import App from './App'
import './index.css'

// Global error handler
window.addEventListener('error', (e) => {
  console.error('[Global Error]', e.error)
  const root = document.getElementById('root')
  if (root) {
    root.innerHTML = `<div style="padding:20px;font-family:sans-serif;">
      <h2>Runtime Error</h2>
      <pre style="background:#f0f0f0;padding:10px;border-radius:8px;overflow:auto;max-height:400px;font-size:12px;">${e.error?.stack || e.message || 'Unknown error'}</pre>
    </div>`
  }
})

window.addEventListener('unhandledrejection', (e) => {
  console.error('[Unhandled Rejection]', e.reason)
})

const root = document.getElementById('root')
if (root) {
  createRoot(root).render(<App />)
} else {
  console.error('Root element not found!')
}
