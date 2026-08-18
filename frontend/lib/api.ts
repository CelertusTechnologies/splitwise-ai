const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080/api/v1";

export type Tokens = {
  access_token: string;
  refresh_token: string;
};

const TOKENS_KEY = "nivra_tokens";

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

export function getTokens(): Tokens | null {
  if (typeof window === "undefined") return null;
  const raw = window.localStorage.getItem(TOKENS_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as Tokens;
  } catch {
    return null;
  }
}

export function setTokens(tokens: Tokens): void {
  window.localStorage.setItem(TOKENS_KEY, JSON.stringify(tokens));
}

export function clearTokens(): void {
  window.localStorage.removeItem(TOKENS_KEY);
}

const PENDING_INVITE_KEY = "nivra_pending_invite";

export function setPendingInvite(code: string): void {
  window.localStorage.setItem(PENDING_INVITE_KEY, code);
}

export function getPendingInvite(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(PENDING_INVITE_KEY);
}

export function clearPendingInvite(): void {
  window.localStorage.removeItem(PENDING_INVITE_KEY);
}

type ApiOptions = {
  token?: string;
  body?: unknown;
};

export async function apiPost<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(options.token ? { Authorization: `Bearer ${options.token}` } : {})
    },
    body: options.body ? JSON.stringify(options.body) : undefined
  });

  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const message = payload?.error?.message ?? "Request failed";
    throw new ApiError(message, response.status);
  }

  return payload as T;
}

export async function apiGet<T>(path: string, token?: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined
  });

  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const message = payload?.error?.message ?? "Request failed";
    throw new ApiError(message, response.status);
  }

  return payload as T;
}

export async function apiDelete<T>(path: string, token?: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: "DELETE",
    headers: token ? { Authorization: `Bearer ${token}` } : undefined
  });

  if (response.status === 204) {
    return undefined as T;
  }

  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const message = payload?.error?.message ?? "Request failed";
    throw new ApiError(message, response.status);
  }

  return payload as T;
}

async function refreshTokens(): Promise<Tokens | null> {
  const tokens = getTokens();
  if (!tokens) return null;

  try {
    const payload = await apiPost<{ data: { tokens: Tokens } }>("/auth/refresh", {
      body: { refresh_token: tokens.refresh_token }
    });
    setTokens(payload.data.tokens);
    return payload.data.tokens;
  } catch {
    clearTokens();
    return null;
  }
}

/**
 * GET a protected endpoint using the stored access token, transparently
 * refreshing and retrying once if the access token has expired.
 */
export async function apiGetAuthed<T>(path: string): Promise<T> {
  const tokens = getTokens();
  if (!tokens) {
    throw new ApiError("Not authenticated", 401);
  }

  try {
    return await apiGet<T>(path, tokens.access_token);
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      const refreshed = await refreshTokens();
      if (refreshed) {
        return await apiGet<T>(path, refreshed.access_token);
      }
    }
    throw err;
  }
}

/**
 * POST to a protected endpoint using the stored access token, transparently
 * refreshing and retrying once if the access token has expired.
 */
export async function apiPostAuthed<T>(path: string, body?: unknown): Promise<T> {
  const tokens = getTokens();
  if (!tokens) {
    throw new ApiError("Not authenticated", 401);
  }

  try {
    return await apiPost<T>(path, { token: tokens.access_token, body });
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      const refreshed = await refreshTokens();
      if (refreshed) {
        return await apiPost<T>(path, { token: refreshed.access_token, body });
      }
    }
    throw err;
  }
}

/**
 * DELETE a protected endpoint using the stored access token, transparently
 * refreshing and retrying once if the access token has expired.
 */
export async function apiDeleteAuthed<T>(path: string): Promise<T> {
  const tokens = getTokens();
  if (!tokens) {
    throw new ApiError("Not authenticated", 401);
  }

  try {
    return await apiDelete<T>(path, tokens.access_token);
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      const refreshed = await refreshTokens();
      if (refreshed) {
        return await apiDelete<T>(path, refreshed.access_token);
      }
    }
    throw err;
  }
}
