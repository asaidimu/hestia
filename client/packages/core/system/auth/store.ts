import { type Transport, type IdentityProvider } from "../../core/client";
import type { LoginResult, ServerHealth } from "./types";

// @note #auth-20261029-003 todo P1 #client,#auth,#ergonomics : HestiaAuth has no session()/restore call
// @author claude
// @see #auth-20261029-001
// @see #auth-20261029-002
// @see #auth-20261029-006
// @status resolved
//
// Resolved: HestiaAuth.session() dispatches system:auth:session:get with notifyAuthStateChange:false; 401 clears identity and returns null, other errors throw. resume() was removed.
// A resume() stub now exists below, but it throws "Method not implemented", so
// the capability is still missing. Proposal: session(): Promise<LoginResult | null>.
//   - dispatches system:auth:session:get with notifyAuthStateChange:false so a
//     "not logged in" answer does not run the global 401 handler;
//   - 200: setIdentity(user) and return { user, expires_at };
//   - 401: clear the local identity and return null;
//   - anything else (network error, 5xx): THROW. Callers must be able to tell
//     "not logged in" from "could not check", or an offline blip logs users out.
// LoginResult gains expires_at?: number (see #auth-20261029-002). Either
// implement resume() on top of session() or replace it outright.

// @note #auth-20261029-004 todo P1 #client,#iam,#integration : @asaidimu/iam 8.2.0 cannot adopt an existing server session
// @author claude
// @see #auth-20261029-003
// @status resolved
//
// Resolved: the reference integration now targets @asaidimu/iam 9.x and uses onRestore to adopt the cookie-backed session.
// To be filed in asaidimu/iam (all verified by reading 8.2.0):
//   1. No startup hook can supply an identity. onVerify only runs when an
//      identity is already persisted; onLoad is undocumented, ignores its
//      boolean result and cannot provide an identity.
//   2. provider.refresh() returns null when there is no identity, so refresh()
//      cannot restore a session.
//   3. onVerify/onLoad are typed Identity | null but onVerify is only called
//      with non-null; onLoad(null) fires twice on a failed verify.
//   4. config.persistence is ignored when scope is passed.
//   5. session.refreshInterval is the lead time before expiry, not a polling
//      interval; this is easy to misread.
// Proposal: onRestore?: () => Promise<(Identity & { session? }) | null>, called
// by the provider whenever the store holds no identity, and by refresh() when
// there is none. Interim, works on 8.2.0: model restore as a variant of
// authenticate(), e.g. authenticate({ kind: "restore" }).

export class HestiaAuth {
  constructor(
    private client: Transport,
    private provider: IdentityProvider,
  ) {}

  async health(): Promise<ServerHealth> {
    const res = await this.client.dispatch<{ data: ServerHealth }>("system:core:health:check");
    return res.data.data;
  }

  // @note #z3g2cp todo : Implement this feature
  // @status resolved
  //
  // Resolved: the resume() stub was replaced by HestiaAuth.session().

  /**
   * Adopt the server's current session, if any. Used on tab boot and when a
   * cross-tab signal says auth changed.
   *
   * Returns `null` when the server says there is no session (401). Throws for
   * anything else (network error, 5xx): callers must be able to tell "not
   * logged in" from "could not check", or an offline blip logs the user out.
   */
  async session(): Promise<LoginResult | null> {
    try {
      const res = await this.client.dispatch<{ data: LoginResult }>(
        "system:auth:session:get",
        { notifyAuthStateChange: false },
      );
      await this.provider.setIdentity(res.data.data.user);
      return res.data.data;
    } catch (err: any) {
      if (isUnauthorized(err)) {
        await this.provider.clear();
        return null;
      }
      throw err;
    }
  }

  async login(email: string, password: string): Promise<LoginResult> {
    const res = await this.client.dispatch<{ data: LoginResult }>(
      "system:auth:session:create",
      { payload: { email, password } },
    );
    await this.provider.setIdentity(res.data.data.user);
    return res.data.data;
  }

  async register(
    email: string,
    password: string,
    name: string,
  ): Promise<{ _id_: string; email: string; name: string }> {
    const res = await this.client.dispatch<{ data: { _id_: string; email: string; name: string } }>(
      "system:users:user:create",
      { payload: { email, password, name } },
    );
    return res.data.data;
  }

  // @note #auth-20261029-005 issue P2 #client,#auth : logout() can leave client and IAM identity diverged; error-code check is dead
  // @author claude
  // @status resolved
  //
  // Resolved: logout() clears local identity in finally and treats an absent/expired session (NO_ACTIVE_SESSION/401) as success, so client and IAM cannot diverge on failure.
  // Any error other than "no active session" is rethrown BEFORE provider.clear(),
  // so a network blip leaves HestiaClient.store.identity set while IAM (which
  // clears first in deauthenticate) is already logged out, and callers that
  // stop the heartbeat after logout() skip that too.
  // err.code === "NO_ACTIVE_SESSION" never matches: DeleteSession returns
  // UNAUTHENTICATED with the message "no active session to revoke"; only the
  // substring check works. With an expired cookie the route (rule
  // authenticated) answers 401 before the handler runs, so that message is
  // never seen at all.
  // Fix: revoke best-effort, clear the local identity in a finally block, and
  // match on code/status rather than message text.

  async logout(): Promise<void> {
    try {
      await this.client.dispatch("system:auth:session:delete");
    } catch (err: any) {
      if (!isNoActiveSession(err)) throw err;
    } finally {
      // Local state must never stay "logged in" because revocation failed: the
      // server session also dies on idle/absolute expiry, and a stale client
      // identity is the worse failure mode.
      await this.provider.clear();
    }
  }

  async requestPasswordReset(email: string): Promise<void> {
    await this.client.dispatch("system:auth:password:reset", { payload: { email } });
  }

  async confirmPasswordReset(
    resetToken: string,
    password: string,
  ): Promise<void> {
    await this.client.dispatch("system:auth:password:confirm", {
      payload: { password, token: resetToken },
    });
  }

  async bootstrap(key: string, password: string, email: string): Promise<void> {
    await this.client.dispatch("system:auth:bootstrap:password:set", {
      payload: { password, email },
      headers: { "X-API-Key": key },
    });
  }
}

/**
 * A 401 from either the real server or the mock transport. The HTTP transport
 * does not preserve the status on SystemError, so match the stable codes too.
 */
function isUnauthorized(err: any): boolean {
  return (
    err?.httpStatus === 401 ||
    err?.code === "UNAUTHORIZED" ||
    err?.code === "UNAUTHENTICATED" ||
    err?.code === "AUTH_REQUIRED" ||
    err?.code === "AUTH-002-UNAUTH"
  );
}

/** Logout is idempotent: an absent/expired session is not a failure. */
function isNoActiveSession(err: any): boolean {
  return (
    err?.code === "NO_ACTIVE_SESSION" ||
    isUnauthorized(err) ||
    err?.message?.includes("no active session")
  );
}
