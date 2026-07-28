import { createRequire } from "node:module";

import tailwindcss from "@tailwindcss/vite";

export default defineNuxtConfig({
  compatibilityDate: "2026-04-01",
  // Disabled: the in-browser devtools panel adds thousands of file
  // watchers, which on macOS exhausts kqueue ("EMFILE: too many open
  // files, watch") or burns 100% CPU when polling is forced. HMR
  // works without it.
  devtools: { enabled: false },
  modules: ["@nuxt/eslint"],
  css: ["~/assets/css/main.css"],
  vite: {
    plugins: [tailwindcss()],
  },
  nitro: {
    externals: {
      // server/utils/temporal.ts loads @temporalio/client with createRequire
      // (see the note there), which hides it from Nitro's dependency tracer.
      // Force-trace it so it still ships in .output/server/node_modules.
      traceInclude: [createRequire(import.meta.url).resolve("@temporalio/client")],
    },
  },
  typescript: {
    strict: true,
    typeCheck: false,
  },
});
