import { HestiaNetworkClient, type IdentityProvider } from "../core/client";
import type { LoginResult, ServerHealth } from "./types";

export class HestiaAuth {
  constructor(
    private client: HestiaNetworkClient,
    private provider: IdentityProvider,
  ) {}

  async health(): Promise<ServerHealth> {
    const res = await this.client.get<{ data: ServerHealth }>("/system/core/health");
    return res.data!.data;
  }

  async login(email: string, password: string): Promise<LoginResult> {
    const res = await this.client.post<{ data: LoginResult }>(
      "/system/auth/session",
      { email, password },
    );
    const result = res.data!.data;
    this.provider.setIdentity(result.user);
    return result;
  }

  async register(
    email: string,
    password: string,
    name: string,
  ): Promise<{ _id_: string; email: string; name: string }> {
    const res = await this.client.post<{
      data: { _id_: string; email: string; name: string, permissions: string[] };
    }>("/system/auth/user", { email, password, name });
    return res.data!.data;
  }

  async logout(): Promise<void> {
    await this.client.delete("/system/auth/session");
    await this.provider.clear();
  }

  async requestPasswordReset(email: string): Promise<void> {
    await this.client.post("/system/auth/password", { email });
  }

  async confirmPasswordReset(
    resetToken: string,
    password: string,
  ): Promise<void> {
    await this.client.patch(
      "/system/auth/password",
      { password, token: resetToken },
    );
  }

  async bootstrap(key: string, password: string, email: string): Promise<void> {
    await this.client.patch(
      "/system/auth/bootstrap",
      { password, email },
      { headers: { "X-API-Key": key } },
    );
  }
}
