// Cliente HTTP que fala com o backend Go Radiant Norma.
//
// Validação 27 (F27.4): tentativa de injetar Authorization header no
// client side foi removida — `rn_jwt` é httpOnly e JS de browser NÃO
// consegue ler `document.cookie` httpOnly (XSS defense). O client não
// tem como adicionar Authorization. Para chamadas autenticadas, use o
// proxy server-side `/v1-api/proxy/[...path]/route.ts` (que LÊ cookie
// via `next/headers cookies()` e adiciona Authorization automaticamente).
//
// Este arquivo permanece para uso em:
//   - Endpoints públicos (sem auth) — ex: /healthz, /readyz
//   - Endpoints server-side (já que Axios pode rodar em node runtime)
//
// Em dev, cookie é setado pelo `/api/login` route que pega X-IF-ID
// direto (compatibilidade com cmd/jwt-mint dev tokens).

import axios, { AxiosError, AxiosInstance } from 'axios'

const API_URL = process.env.RADIANT_API_URL || 'http://localhost:8080'

export const api: AxiosInstance = axios.create({
  baseURL: API_URL,
  timeout: 30_000,
  // CSRF: backend aceita bearer-only (sem cookie-based auth).
  withCredentials: false,
})

// Centralized error handler — backend returns {error: "..."}.
api.interceptors.response.use(
  (response) => response,
  (error: AxiosError<{ error?: string }>) => {
    const message = error.response?.data?.error ?? error.message ?? 'erro desconhecido'
    // In dev, surface to console.
    if (process.env.NODE_ENV !== 'production') {
      console.error(`[API ${error.config?.method?.toUpperCase()} ${error.config?.url}]`, message)
    }
    return Promise.reject(new Error(message))
  }
)
