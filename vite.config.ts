import tailwindcss from "@tailwindcss/vite";
import adapter from "@sveltejs/adapter-auto";
import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [
    tailwindcss(),
    sveltekit({
      compilerOptions: {
        runes: ({ filename }) =>
          filename.split(/[/\\]/).includes("node_modules") ? undefined : true,
      },
      adapter: adapter(),
    }),
  ],
  server: {
    host: "0.0.0.0",
    allowedHosts: ["fgbacademy.rybuildstuff.dev"],

    port: 5173,
    hmr: {
      clientPort: 3038,
    },
    // On Windows + Docker Desktop, host file-change events do not propagate
    // into the Linux container via inotify. Polling makes Vite detect edits
    // to bind-mounted source so HMR works.
    watch: {
      usePolling: true,
      interval: 300,
    },
    proxy: {
      "/api": {
        target: "http://backend:5555",
        changeOrigin: true,
      },
    },
  },
});
