// Cliente HTTP que fala com o backend Go Radiant Norma.
//
// JWT bearer token via cookie httpOnly (setado pelo NextAuth.js após
// login). Axios intercepta todas as requests e adiciona header
// `Authorization: Bearer <token>` para backend.
//
// Em dev, cookie é setado pelo `/api/login` route que pega X-IF-ID
// direto (compatibilidade com cmd/jwt-mint dev tokens).

import axios, { AxiosError, AxiosInstance } from 'axios'
import { getCookie } from './cookies'

const API_URL = process.env.RADIANT_API_URL || 'http://localhost:8080'

export const api: AxiosInstance = axios.create({
  baseURL: API_URL,
  timeout: 30_000,
  // CSRF: backend aceita bearer-only (sem cookie-based auth).
  withCredentials: false,
})

// Inject bearer token from httpOnly cookie (or localStorage fallback).
api.interceptors.request.use((config) => {
  const token = getCookie('rn_jwt')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
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
