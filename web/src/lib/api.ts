function getApiBase(): string {
  return import.meta.env.VITE_API_BASE_URL ?? ''
}

export function apiPath(path: string): string {
  return getApiBase() + '/api' + path
}

export async function authorizedJsonFetch(
  input: RequestInfo | URL,
  getToken: () => Promise<string | null>,
  init: RequestInit = {},
): Promise<Response> {
  const token = await getToken()
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  return fetch(input, {
    ...init,
    headers,
  })
}
