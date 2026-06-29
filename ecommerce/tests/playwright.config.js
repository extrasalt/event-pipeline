import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  timeout: 30000,
  retries: 0,
  use: {
    baseURL: "http://localhost:8080",
    headless: true,
  },
  webServer: {
    command: "go run .",
    port: 8080,
    cwd: "..",
    reuseExistingServer: true,
  },
});
