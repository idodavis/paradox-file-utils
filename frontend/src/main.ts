/**
 * Vue app bootstrap.
 * Registers Nuxt UI, Paradox syntax highlighting, and mounts the app.
 */
import { createApp } from "vue";
import ui from "@nuxt/ui/vue-plugin";
import "./styles/nuxt-ui.css";
import App from "./App.vue";
import { createPinia } from "pinia";
import router from "./router";
import { registerParadoxLanguages } from "./composables/registerSyntax";

void registerParadoxLanguages().then(() => {
  const app = createApp(App);
  app.use(createPinia());
  app.use(router);
  app.use(ui);
  app.mount("#app");
});
