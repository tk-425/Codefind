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
