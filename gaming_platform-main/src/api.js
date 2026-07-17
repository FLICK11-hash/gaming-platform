// In dev, Vite proxies '/api' to localhost:8080 (see vite.config.js).
// In production, set VITE_API_URL to your Render backend URL, e.g.
// https://gaming-platform-api.onrender.com — Vercel has no idea where
// your Go server lives otherwise.
const API_BASE = import.meta.env.VITE_API_URL ? `${import.meta.env.VITE_API_URL}/api` : '/api'

export function api(path, { token, method = 'GET', body } = {}) {
  const headers = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`
  return fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  }).then(async res => {
    const text = await res.text()
    const data = text ? JSON.parse(text) : null
    if (!res.ok) throw new Error(data?.error || data?.message || `${res.status} ${res.statusText}`)
    return data
  })
}
