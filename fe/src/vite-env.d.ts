/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Bearer token，与 rd 启动的 WBFLOW_AUTH_TOKEN 同值；空=请求不带 Authorization（后端 fail closed 全 401） */
  readonly VITE_WBFLOW_AUTH_TOKEN?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
