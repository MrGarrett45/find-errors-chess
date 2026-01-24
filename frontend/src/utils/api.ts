type TokenProvider = (options?: { authorizationParams?: { audience?: string } }) => Promise<string>;

const audience = import.meta.env.VITE_AUTH0_AUDIENCE as string | undefined;
const API_BASE =
  import.meta.env.VITE_API_BASE_URL?.replace(/\/$/, '') || 'http://localhost:8080';

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

export async function createCheckoutSession(
  token: string,
): Promise<{ url: string }> {
  const res = await fetch(`${API_BASE}/api/billing/create-checkout-session`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    throw new Error(`Failed to create checkout session: ${res.status}`);
  }

  return res.json();
}

export async function updatePlan(
  token: string,
  plan: 'PRO' | 'FREE',
): Promise<{ status: string }> {
  const res = await fetch(`${API_BASE}/api/billing/update-plan`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ plan }),
  });

  if (!res.ok) {
    throw new Error(`Failed to update plan: ${res.status}`);
  }

  return res.json();
}

export async function createBillingPortalSession(
  token: string,
): Promise<{ url: string }> {
  const res = await fetch(`${API_BASE}/api/billing/portal-session`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    throw new Error(`Failed to create billing portal session: ${res.status}`);
  }

  return res.json();
}
