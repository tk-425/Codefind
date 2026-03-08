const LOCAL_CALLBACK_HOST = '127.0.0.1'
const LOCAL_CALLBACK_PATH = '/callback'
const CLI_REDIRECT_STORAGE_KEY = 'codefind.cli_redirect_uri'

export function getPostAuthPath(orgId: string | null): string {
  return orgId ? '/search' : '/no-access'
}

export function getCliRedirectUri(search: URLSearchParams): string | null {
  const redirectUri = search.get('redirect_uri')
  if (!redirectUri) {
    return null
  }
  return isCliRedirectUri(redirectUri) ? redirectUri : null
}

export function loadStoredCliRedirectUri(): string | null {
  const storedValue = window.sessionStorage.getItem(CLI_REDIRECT_STORAGE_KEY)
  if (!storedValue) {
    return null
  }
  return isCliRedirectUri(storedValue) ? storedValue : null
}

export function storeCliRedirectUri(value: string): void {
  if (isCliRedirectUri(value)) {
    window.sessionStorage.setItem(CLI_REDIRECT_STORAGE_KEY, value)
  }
}

export function clearStoredCliRedirectUri(): void {
  window.sessionStorage.removeItem(CLI_REDIRECT_STORAGE_KEY)
}

export function isCliRedirectUri(value: string): boolean {
  try {
    const parsed = new URL(value)
    return (
      parsed.protocol === 'http:' &&
      parsed.hostname === LOCAL_CALLBACK_HOST &&
      parsed.pathname === LOCAL_CALLBACK_PATH &&
      parsed.port.length > 0
    )
  } catch {
    return false
  }
}

export async function postCliToken(redirectUri: string, token: string): Promise<void> {
  const response = await fetch(redirectUri, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ token }),
  })
  if (!response.ok) {
    throw new Error(`CLI callback failed with status ${response.status}`)
  }
}
