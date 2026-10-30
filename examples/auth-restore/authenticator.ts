import { HestiaClient, UserIdentity } from "@asaidimu/hestia";
import { createAuthenticator, type Identity } from "@asaidimu/iam";
import { ArtifactContainer } from "@asaidimu/utils-artifacts";
import { ReactiveDataStore } from "@asaidimu/utils-store";

// @note #auth-20261029-010 context #auth,#client,#integration,#example : Reference integration of HestiaClient with @asaidimu/iam 8.2.0
// @author claude
// @see #auth-20261029-001
// @see #auth-20261029-002
// @see #auth-20261029-003
// @see #auth-20261029-004
// @see #auth-20261029-006
// @see #auth-20261029-007
// @see #auth-20261029-008
// @see #auth-20261029-009
//
// Principle: the server's HttpOnly session cookie is the single source of
// truth. IAM holds a copy of the claims for the UI, and every path that could
// make that copy stale re-asks the server instead of trusting memory:
//   - tab boot                 -> restore() adopts an existing session
//   - tab regains focus        -> refresh() re-validates and re-fetches claims
//   - another tab logs in/out  -> BroadcastChannel ping -> re-ask the server
//   - any request gets a 401   -> client.onAuthStateChange -> refresh()
//   - identity set or cleared  -> heartbeat starts or stops (the heartbeat
//     keeps the sliding server session alive: 30 min idle limit, renewed by
//     any authenticated request)
//
// Written against the API specified by the notes above, which does not exist
// yet: client.auth.session() resolving { user, expires_at? } or null on 401
// and THROWING on network/5xx (unknown, so the user must not be logged out),
// and expires_at (unix seconds) on login results. HestiaAuth.resume() is the
// current placeholder and throws "Method not implemented"; treat it as the
// interim name for session(). This file is not part of any package build; it
// type-checked against @asaidimu/iam 8.2.0 declarations with a stub for the
// missing client methods.

type AuthRequest =
  | { kind: "login"; email: string; password: string }
  | { kind: "restore" };

type Claims = { user: UserIdentity; expires_at?: number };

const CHANNEL_NAME = "hestia:auth";

// Used only when the server does not send expires_at (older servers).
const FALLBACK_SESSION = { ttl: 8 * 60 * 60 * 1000, refreshInterval: 2 * 60 * 1000 };

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
  // @note #auth-20261029-013 todo P3 #auth,#client,#integration : Move cross-tab signalling and the 401 bridge into HestiaClient
  // @author claude
  // @see #auth-20261029-007
  // @see #auth-20261029-009
  //
  // The BroadcastChannel ping, the visibilitychange refresh and the
  // client.onAuthStateChange(() => refresh()) bridge below exist only because
  // HestiaClient neither broadcasts auth changes nor offers a real
  // subscription. Never broadcast claims, only "changed"; never post from the
  // handler of a received message (ping-pong).
  const channel =
    typeof BroadcastChannel !== "undefined" ? new BroadcastChannel(CHANNEL_NAME) : null;
  const cleanups: Array<() => void> = [];

  const auth = await createAuthenticator<UserIdentity, AuthRequest, any>({
    onAuthenticate: async (request) => {
      // @note #auth-20261029-011 todo P2 #auth,#iam,#integration : Remove the restore request kind once iam gains onRestore
      // @author claude
      // @see #auth-20261029-004
      //
      // iam 8.2.0 has no startup hook that can supply an identity, so restore
      // is modelled as a variant of authenticate(). Replace this branch with
      // onRestore: () => client.auth.session() and drop AuthRequest.
      if (request.kind === "restore") {
        // null (no session) makes authenticate() resolve false, no error.
        return resultOf(await client.auth.session());
      }
      return resultOf(await client.auth.login(request.email, request.password));
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
    // and revocations reach the UI. null (401) logs out; a thrown error means
    // "could not check", so keep the user in: the server still enforces.
    onRefresh: async (current) => {
      try {
        return resultOf(await client.auth.session());
      } catch {
        return current;
      }
    },

    // Cross-tab: another tab logged in or out. Re-ask the server rather than
    // trusting any payload, then hand IAM the outcome.
    onAuthChange: (notify) => {
      if (!channel) return () => {};
      const onMessage = async () => {
        try {
          const claims = await client.auth.session();
          notify(claims && resultOf(claims) ? identityOf(claims) : null);
        } catch {
          // Offline or server error: keep the current state.
        }
      };
      channel.addEventListener("message", onMessage);
      return () => channel.removeEventListener("message", onMessage);
    },

    // Fallback only; the server's expires_at overrides it per login/refresh.
    session: FALLBACK_SESSION,

    scope: {
      container: container.scope("iam") as any,
      store: store,
    },
  });

  // @note #auth-20261029-012 todo P2 #auth,#client,#integration : Delete this heartbeat bridge once HestiaClient owns the heartbeat lifecycle
  // @author claude
  // @see #auth-20261029-008
  //
  // The heartbeat follows the identity, whatever changed it (login, restore,
  // refresh, expiry, cross-tab), and it is what slides the server session.
  // HestiaClient should start and stop it itself when its identity changes.
  cleanups.push(
    auth.onChange((identity) => {
      if (identity) client.startHeartbeat();
      else client.stopHeartbeat();
    }),
  );

  // Any 401 from any request: let IAM re-validate and clear if needed.
  client.onAuthStateChange(() => {
    void auth.refresh();
  });

  // Back in the foreground (tab switch, laptop wake): re-validate now rather
  // than showing stale state until the next timer tick.
  if (typeof document !== "undefined") {
    const onVisible = () => {
      if (document.visibilityState === "visible") void auth.refresh();
    };
    document.addEventListener("visibilitychange", onVisible);
    cleanups.push(() => document.removeEventListener("visibilitychange", onVisible));
  }

  const announce = () => channel?.postMessage("changed");

  /** Adopt a session that already exists on the server. Never throws. */
  const restore = async (): Promise<boolean> => {
    try {
      return await auth.authenticate({ kind: "restore" });
    } catch (e) {
      console.warn("[auth] could not check for an existing session", e);
      return false;
    }
  };

  /** Throws on bad credentials; tells other tabs on success. */
  const login = async (email: string, password: string): Promise<boolean> => {
    const ok = await auth.authenticate({ kind: "login", email, password });
    if (ok) announce();
    return ok;
  };

  /** Prefer this over auth.deauthenticate(): it also tells other tabs. */
  const logout = async (): Promise<boolean> => {
    const ok = await auth.deauthenticate();
    announce();
    return ok;
  };

  const destroy = auth.destroy.bind(auth);

  // Resolve the initial state before the app renders, so a user who already
  // has a session in another tab never sees a login screen.
  await restore();

  return Object.assign(auth, {
    login,
    logout,
    restore,
    destroy: async () => {
      cleanups.forEach((dispose) => dispose());
      channel?.close();
      await destroy();
    },
  });
}
