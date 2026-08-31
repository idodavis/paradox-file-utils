/**
 * Vue app bootstrap with Pinia, router, and Nuxt UI.
 */
import { createApp } from "vue";
import ui from "@nuxt/ui/vue-plugin";
import "./styles/nuxt-ui.css";
import App from "./App.vue";
import { createPinia } from "pinia";
import { PiniaColada } from "@pinia/colada";
import router from "./router";
import { applySeedCss } from "./ide/colorThemes";

applySeedCss("pmt-dark");

const app = createApp(App);
app.use(createPinia());
app.use(PiniaColada, {
  queryOptions: {
    staleTime: 30_000,
    refetchOnWindowFocus: false,
  },
});
app.use(router);
app.use(ui);
app.mount("#app");
