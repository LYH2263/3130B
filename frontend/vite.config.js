import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5130,
    proxy: {
      '/api': {
        target: 'http://localhost:8130',
        changeOrigin: true,
        ws: true,
      },
    },
  },
});
