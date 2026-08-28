/// <reference types="vite/client" />

/**
 * Vue SFC and grammar module declarations.
 */
declare module "*.vue" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<Record<string, never>, Record<string, never>, any>;
  export default component;
}

declare module "*.tmLanguage.json" {
  const grammar: Record<string, unknown>;
  export default grammar;
}

declare module "*?worker" {
  const workerConstructor: new () => Worker;
  export default workerConstructor;
}

declare module "*?url" {
  const url: string;
  export default url;
}
