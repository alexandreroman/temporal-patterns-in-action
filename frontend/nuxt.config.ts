import tailwindcss from "@tailwindcss/vite";

export default defineNuxtConfig({
  compatibilityDate: "2026-04-01",
  devtools: { enabled: true },
  modules: ["@nuxt/eslint"],
  css: ["~/assets/css/main.css"],
  vite: {
    plugins: [tailwindcss()],
  },
  runtimeConfig: {
    natsUrl: process.env.NATS_URL ?? "nats://localhost:4222",
  },
  typescript: {
    strict: true,
    typeCheck: false,
  },
});
