import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Configuracion de Vite.
//
// El proxy de /api hacia el backend existe para que en desarrollo el navegador
// vea un solo origen (localhost:5173) y no haya CORS de por medio. En produccion
// (fase 8) el bundle lo sirve el mismo servidor que la API, asi que la ruta
// relativa /api/v1 tambien funciona sin tocar nada.
//
// VITE_API_URL permite apuntar a otro backend sin editar este archivo; si no
// esta, se asume el de desarrollo.
export default defineConfig(() => ({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
}))
