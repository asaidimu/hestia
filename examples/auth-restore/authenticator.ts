import { HestiaClient, type UserIdentity } from "@asaidimu/hestia";
import { createAuthenticator, type Identity } from "@asaidimu/iam";
import { ArtifactContainer } from "@asaidimu/utils-artifacts";
import { ReactiveDataStore } from "@asaidimu/utils-store";

// @note #auth-20261029-010 context : Reference integration of HestiaClient with @asaidimu/iam 9.x
// @tags #auth #client #integration #example
// @author claude
// @see #auth-20261029-001
// @see #auth-20261029-002
// @see #auth-20261029-003
// @see #auth-20261029-004
// @see #auth-20261029-006
// @see #auth-20261029-007
// @see #auth-20261029-008
// @see #auth-20261029-009
// @status resolved
//
// Resolved: reference integration updated to @asaidimu/iam 9.x (async factory, onRestore, refreshResult, onAuthChange mirroring HestiaClient).
// Principle: the server's HttpOnly session cookie is the single source of
// truth. IAM holds a copy of the claims for the UI, and every path that could
// make that copy stale re-asks the server instead of trusting memory:
//   - tab boot                 -> onRestore adopts an existing session
//   - tab regains focus        -> refresh() re-validates and re-fetches claims
//   - another tab logs in/out  -> HestiaClient BroadcastChannel -> auth.session()
//   - any request gets a 401   -> HestiaClient clears identity -> IAM mirrors it
//   - identity set or cleared  -> HestiaClient starts/stops the heartbeat
//
// Requires @asaidimu/iam >= 9 (onRestore, async createAuthenticator,
// refreshResult, refreshLeadTime) and HestiaAuth.session() on the client.
// This file is not part of any package build.

type Credentials = { email: string; password: string };

type Claims = { user: UserIdentity; expires_at?: number };

// Used only when the server does not send expires_at (older servers).
const FALLBACK_SESSION = {
  ttl: 8 * 60 * 60 * 1000,
  refreshLeadTime: 2 * 60 * 1000,
};

/** Plain IAM identity: what the UI reads and what is shared across tabs. */
function identityOf({ user }: Claims): Identity<UserIdentity, any> {
  return { permissions: (user as any).permissions ?? [], properties: user };
}

/**
 * IAM authenticate/refresh result: the identity plus a per-login session TTL
 * taken from the server's absolute expiry, so the client never outlives or
 * undershoots the real session. Returns null when the session has already
 * expired.
 */
function resultOf(claims: Claims | null) {
  if (!claims) return null;
  if (claims.expires_at === undefined) return identityOf(claims);
  const ttl = claims.expires_at * 1000 - Date.now();
  return ttl > 0 ? { ...identityOf(claims), session: { ttl } } : null;
}

export async function Authenticator({
  client,
  container,
  store,
}: {
  client: HestiaClient;
  container: ArtifactContainer;
  store: ReactiveDataStore<any>;
}) {
  const cleanups: Array<() => void> = [];

  const auth = await createAuthenticator<UserIdentity, Credentials, any>({
    // Adopt a cookie-backed session that already exists on the server. This is
    // what makes a fresh tab stay logged in; `null` means "no session".
    onRestore: async () => {
      const claims = await client.auth.session();
      return claims ? identityOf(claims) : null;
    },

    onAuthenticate: async (credentials) => {
      return resultOf(
        await client.auth.login(credentials.email, credentials.password),
      );
    },

    onDeauthenticate: async () => {
      try {
        await client.auth.logout();
      } catch (e) {
        // Best effort: the local state is cleared regardless, and the server
        // session also dies on idle/absolute expiry.
        console.warn("[auth] server logout failed", e);
      }
      return true;
    },

    // Re-fetch claims instead of echoing the old ones, so permission changes
    // and revocations reach the UI. `null` (401) logs out. A thrown error is
    // reported as `unavailable` and keeps the last known identity, so do not
    // catch it here: the server still enforces authority.
    onRefresh: async () => {
      return resultOf(await client.auth.session());
    },

    // Runs once after startup, with the final restored identity (or null).
    onLoad: async (identity) => {
      console.info(
        "[auth] startup complete",
        identity ? "authenticated" : "anonymous",
      );
    },

    // HestiaClient is the authority for cross-tab changes: it broadcasts only
    // "changed" and re-asks the server. Mirror its store into IAM instead of
    // running a second BroadcastChannel here.
    onAuthChange: (notify) => {
      return client.onAuthStateChange((identity) => {
        notify(
          identity
            ? { permissions: (identity as any).permissions ?? [], properties: identity }
            : null,
        );
      });
    },

    // Fallback only; expires_at from login/refresh overrides it per session.
    // IAM's onRestore cannot carry a per-restore TTL yet, so a restored
    // session uses this value until the first refresh.
    session: FALLBACK_SESSION,

    scope: {
      container: container.scope("iam") as any,
      store: store,
    },
  });

  // HestiaClient owns the heartbeat now; no bridge is needed here.

  // Back in the foreground (tab switch, laptop wake): re-validate now rather
  // than showing stale state until the next timer tick.
  if (typeof document !== "undefined") {
    const onVisible = () => {
      if (document.visibilityState === "visible") void auth.refresh();
    };
    document.addEventListener("visibilitychange", onVisible);
    cleanups.push(() =>
      document.removeEventListener("visibilitychange", onVisible),
    );
  }

  /** Adopt a session that already exists on the server. Never throws. */
  const restore = async (): Promise<boolean> => {
    try {
      return (await auth.restore()) !== null;
    } catch (e) {
      console.warn("[auth] could not check for an existing session", e);
      return false;
    }
  };

  /** Throws on bad credentials. Other tabs are told by HestiaClient. */
  const login = async (email: string, password: string): Promise<boolean> => {
    return auth.authenticate({ email, password });
  };

  /** Prefer this over auth.deauthenticate(): it clears the server session. */
  const logout = async (): Promise<boolean> => {
    return auth.deauthenticate();
  };

  const destroy = auth.destroy.bind(auth);

  // createAuthenticator already ran onRestore before returning, so the initial
  // state is resolved before the app renders. The exported `restore` remains
  // available for an explicit re-check.

  return Object.assign(auth, {
    login,
    logout,
    restore,
    destroy: async () => {
      cleanups.forEach((dispose) => dispose());
      await destroy();
    },
  });
}
