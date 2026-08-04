/**
 * Frontend bootstrap entry.
 * Switches between legacy Svelte and migration Vue app by Vite mode.
 */
import "../style.css";

const framework = import.meta.env.MODE === "vue" ? "vue" : "svelte";

if (framework === "vue") {
  await import("../src-vue/main");
} else {
  await import("./main");
}
