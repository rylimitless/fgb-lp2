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
    port: 5173,
    allowedHosts: [".ngrok-free.app", "fgbguide.rybuildstuff.dev"],
    hmr: {
      clientPort: 5173,
    },
    proxy: {
      "/api": {
        target: "http://backend:5555",
        changeOrigin: true,
      },
    },
  },
});
