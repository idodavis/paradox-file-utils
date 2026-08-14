/**
 * Vite config: Vue, Nuxt UI, Wails, Tailwind, monaco-vscode workers/CSS.
 */
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import ui from "@nuxt/ui/vite";
import wails from "@wailsio/runtime/plugins/vite";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

export default defineConfig({
  plugins: [
    vue(),
    ui({ colorMode: false }),
    wails("./bindings"),
    tailwindcss(),
    {
      name: "load-vscode-css-as-string",
      enforce: "pre",
      async resolveId(source, importer, options) {
        const resolved = await this.resolve(source, importer, options);
        if (
          resolved &&
          resolved.id.match(
            /node_modules\/(@codingame\/monaco-vscode|vscode|monaco-editor).*\.css$/,
          )
        ) {
          return { ...resolved, id: resolved.id + "?inline" };
        }
        return undefined;
      },
    },
  ],
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  worker: { format: "es" },
  build: { target: "esnext" },
  optimizeDeps: {
    include: [
      "vscode-textmate",
      "vscode-oniguruma",
      "@vscode/vscode-languagedetection",
      "marked",
      "vscode/localExtensionHost",
    ],
  },
  resolve: {
    alias: {
      "@services": path.resolve(
        import.meta.dirname,
        "bindings/paradox-modding-tools/services",
      ),
      "@assets": path.resolve(import.meta.dirname, "src/assets"),
    },
    dedupe: ["vscode", "monaco-editor"],
  },
});
