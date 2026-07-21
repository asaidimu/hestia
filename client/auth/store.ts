import { type Transport, type IdentityProvider } from "../core/client";
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
    this.provider.setIdentity(res.data.data.user);
    return res.data.data;
  }

  async register(
    email: string,
    password: string,
    name: string,
  ): Promise<{ _id_: string; email: string; name: string }> {
    const res = await this.client.dispatch<{ data: { _id_: string; email: string; name: string } }>(
      "system:auth:user:register",
      { payload: { email, password, name } },
    );
    return res.data.data;
  }

  async logout(): Promise<void> {
    await this.client.dispatch("system:auth:session:delete");
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
