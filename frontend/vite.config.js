import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  base: "./",
  build: {
    outDir: "dist",
    target: "safari17",
    cssCodeSplit: false,
    reportCompressedSize: false,
  },
  server: {
    port: 5174,
  },
});
