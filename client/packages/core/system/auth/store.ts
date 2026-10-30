import { type Transport, type IdentityProvider } from "../../core/client";
import type { LoginResult, ServerHealth } from "./types";

// @note #auth-20261029-003 todo P1 #client,#auth,#ergonomics : HestiaAuth has no session()/restore call
// @author claude
// @see #auth-20261029-001
// @see #auth-20261029-002
// @see #auth-20261029-006
//
// Proposal: session(): Promise<LoginResult | null>.
//   - dispatches system:auth:session:get with notifyAuthStateChange:false so a
//     "not logged in" answer does not run the global 401 handler;
//   - 200: setIdentity(user) and return { user, expires_at };
//   - 401: clear the local identity and return null;
//   - anything else (network error, 5xx): THROW. Callers must be able to tell
//     "not logged in" from "could not check", or an offline blip logs users out.
// LoginResult gains expires_at?: number (see #auth-20261029-002).

// @note #auth-20261029-004 todo P1 #client,#iam,#integration : @asaidimu/iam 8.2.0 cannot adopt an existing server session
// @author claude
// @see #auth-20261029-003
//
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

  async resume():Promise<LoginResult>{
      // Useful for syncing state when a user opens a
      // second tab
      throw new Error("Method not implemented!")
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
  //
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
      if (err?.code === "NO_ACTIVE_SESSION" || err?.message?.includes("no active session")) {
        // logout is idempotent — no session to revoke is fine
      } else {
        throw err;
      }
    }
    await this.provider.clear();
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
