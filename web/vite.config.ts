import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  root: "web",
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:3000",
      "/_": "http://127.0.0.1:3000",
    },
  },
  build: {
    outDir: "../internal/web/dist",
    emptyOutDir: true,
    sourcemap: false,
  },
});
