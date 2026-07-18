import { HestiaNetworkClient, type IdentityProvider } from "../core/client";
import type { LoginResult, ServerHealth, TokenPair } from "./types";

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
    this.client.storeTokens(result.token.access, result.token.refresh);
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

  async refresh(refreshToken?: string): Promise<TokenPair> {
    const body = refreshToken ? { refresh_token: refreshToken } : {};
    const res = await this.client.patch<{ data: { token: TokenPair } }>(
      "/system/auth/session",
      body,
    );
    return res.data!.data.token;
  }

  async logout(): Promise<void> {
    const refresh = this.provider.token("refresh");
    const body = refresh ? { refresh_token: refresh } : {};
    await this.client.delete("/system/auth/session", body);
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
      { password, token:resetToken },
      { headers: { Authorization: `Bearer ${resetToken}` } },
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
