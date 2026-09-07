import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

// rd 后端地址经 WBFLOW_API_PORT 覆盖端口（默认 8080，见 rd/configs/config.yaml http.port）。
const apiPort = process.env.WBFLOW_API_PORT ?? "8080";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": {
        target: `http://127.0.0.1:${apiPort}`,
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: "./src/test-setup.ts",
    globals: true,
  },
});
