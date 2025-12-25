type TokenProvider = (options?: { authorizationParams?: { audience?: string } }) => Promise<string>;

const audience = import.meta.env.VITE_AUTH0_AUDIENCE as string | undefined;

export async function authFetch(
  input: RequestInfo | URL,
  init: RequestInit | undefined,
  getToken: TokenProvider,
) {
  const token = await getToken(
    audience ? { authorizationParams: { audience } } : undefined,
  );
  const headers = new Headers(init?.headers);
  headers.set('Authorization', `Bearer ${token}`);
  return fetch(input, { ...init, headers });
}
