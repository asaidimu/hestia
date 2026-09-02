import { type Transport, type IdentityProvider } from "../../core/client";
import type { LoginResult, ServerHealth } from "./types";

export class HestiaAuth {
  constructor(
    private client: Transport,
    private provider: IdentityProvider,
  ) {}

  async health(): Promise<ServerHealth> {
    const res = await this.client.dispatch<{ data: ServerHealth }>("system:core:health:check");
    return res.data.data;
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
