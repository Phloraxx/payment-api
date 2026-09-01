import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  root: "web-v4",
  plugins: [react()],
  server: {
    port: 5174,
    proxy: { "/admin": "http://127.0.0.1:8091", "/health": "http://127.0.0.1:8091" },
  },
  build: {
    outDir: "../internal/v4web/dist",
    emptyOutDir: true,
    sourcemap: false,
  },
});
