import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import vue from "@vitejs/plugin-vue";
import ui from "@nuxt/ui/vite";
import wails from "@wailsio/runtime/plugins/vite";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const useVue = mode === "vue";

  return {
    plugins: [
      useVue ? vue() : svelte(),
      ...(useVue
        ? [
            ui({
              colorMode: false,
            }),
          ]
        : []),
      wails("./bindings"),
      tailwindcss(),
    ],
    server: {
      host: "127.0.0.1",
      port: Number(process.env.WAILS_VITE_PORT) || 9245,
      strictPort: true,
    },
    resolve: {
      alias: {
        "@services": path.resolve(import.meta.dirname, "bindings/paradox-modding-tools/services"),
        "@components": path.resolve(import.meta.dirname, "src/lib/components"),
        "@pages": path.resolve(import.meta.dirname, "src/lib/pages"),
        "@stores": path.resolve(import.meta.dirname, "src/lib/stores"),
        "@assets": path.resolve(import.meta.dirname, "src/assets"),
        "@utils": path.resolve(import.meta.dirname, "src/lib/utils"),
      },
    },
  };
});
