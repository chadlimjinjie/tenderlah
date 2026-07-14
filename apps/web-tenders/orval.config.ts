import { defineConfig } from "orval"

export default defineConfig({
  gatewayApi: {
    input: {
      target: "./lib/api/gateway-openapi.json",
    },
    output: {
      mode: "tags-split",
      target: "./lib/api/generated",
      schemas: "./lib/api/generated/model",
      client: "react-query",
      httpClient: "axios",
      override: {
        mutator: {
          path: "./lib/api/axios-instance.ts",
          name: "customAxiosInstance",
        },
      },
    },
  },
})
