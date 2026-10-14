import Axios from "axios"

import { getAccessToken } from "@/lib/auth/oidc"

export const API_BASE = process.env.NEXT_PUBLIC_GATEWAY_URL ?? "http://localhost:8080"

export const axiosInstance = Axios.create({ baseURL: API_BASE })

// Backend services authenticate with the auth-service's OIDC access token.
axiosInstance.interceptors.request.use(async (config) => {
  const token = await getAccessToken()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

export const customAxiosInstance = <T>(
  config: Parameters<typeof axiosInstance.request>[0]
): Promise<T> => {
  return axiosInstance.request<T>(config).then(({ data }) => data)
}
