import { defineConfig } from "vite";

export default defineConfig({
  build: {
    outDir: "dist",
    rollupOptions: {
      input: {
        main: "index.html",
        quest: "quest.html",
        progress: "progress.html",
      },
    },
  },
  server: { proxy: { "/api": "http://localhost:8080" } }, // dev only
});
