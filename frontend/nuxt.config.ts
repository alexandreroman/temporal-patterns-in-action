import tailwindcss from "@tailwindcss/vite";

if (process.platform === "darwin" && !process.env.CHOKIDAR_USEPOLLING) {
  process.env.CHOKIDAR_USEPOLLING = "true";
}

export default defineNuxtConfig({
  compatibilityDate: "2026-04-01",
  devtools: { enabled: true },
  modules: ["@nuxt/eslint"],
  css: ["~/assets/css/main.css"],
  vite: {
    plugins: [tailwindcss()],
    server: {
      watch: {
        usePolling: process.platform === "darwin",
        interval: 300,
      },
    },
  },
  typescript: {
    strict: true,
    typeCheck: false,
  },
});
