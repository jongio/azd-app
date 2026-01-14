interface ImportMetaEnv {
  readonly MODE: string
  readonly DEV?: boolean
  readonly PROD?: boolean
  // add other Vite env vars your app uses, e.g. VITE_API_URL
  readonly VITE_API_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
