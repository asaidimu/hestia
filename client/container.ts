import type { SimplePersistence } from "@asaidimu/utils-persistence";
import { ReactiveDataStore } from "@asaidimu/utils-store";
import { HestiaAuth } from "./auth/store";
import { HestiaCollections } from "./collections/store";
import {
    HttpTransport,
    type Transport,
    type IdentityProvider,
} from "./core/client";
import { WailsTransport } from "./core/wails-transport";
import { HestiaKeyStore } from "./system/api-keys/store";
import { HestiaUsers } from "./system/identity/store";
import type { UserIdentity } from "./system/identity/types";
import { HestiaLogs } from "./system/logs/store";
import { HestiaPolicies } from "./system/policies/store";
import { HestiaRules } from "./system/rules/store";
import { HestiaBlobClient } from "./blobs/store";
import { HestiaCapabilities } from "./system/capabilities/store";

export interface HestiaConfig {
  baseUrl: string;
  apiPrefix?: string;
  persistence?: SimplePersistence<AuthState>;
  transport?: Transport;
}

interface AuthState {
  identity: UserIdentity | null;
}

export class HestiaClient {
  readonly store: ReactiveDataStore<AuthState>;
  readonly client: Transport;
  readonly auth: HestiaAuth;
  readonly users: HestiaUsers;
  readonly keys: HestiaKeyStore;
  readonly policies: HestiaPolicies;
  readonly rules: HestiaRules;
  readonly logs: HestiaLogs;
  readonly collections: HestiaCollections;
  readonly blobs: HestiaBlobClient;
  readonly capabilities: HestiaCapabilities
  private tokenProvider: IdentityProvider;

  private onAuthStateChanged?: () => void;

  constructor(config: HestiaConfig) {
    this.store = new ReactiveDataStore<AuthState>(
      { identity: null },
      config.persistence,
    );

    const tokenProvider: IdentityProvider = {
      identity: () => this.store.get().identity,
      setIdentity: async (identity) =>
        void (await this.store.set({ identity })),
      clear: async () =>
        void (await this.store.set({ identity: null })),
    };

    const apiPrefix = config.apiPrefix ?? "/api";

    const onUnauthorized = () => {
      tokenProvider.clear();
      this.onAuthStateChanged?.();
    };

    this.tokenProvider = tokenProvider;
    if (config.transport instanceof WailsTransport) {
      config.transport.configure(config.baseUrl, apiPrefix, tokenProvider);
      config.transport.setOnUnauthorized(onUnauthorized);
      this.client = config.transport;
    } else {
      this.client = config.transport ?? new HttpTransport(
        config.baseUrl,
        apiPrefix,
        onUnauthorized,
      );
    }

    this.auth = new HestiaAuth(this.client, tokenProvider);
    this.users = new HestiaUsers(this.client);
    this.keys = new HestiaKeyStore(this.client,);
    this.policies = new HestiaPolicies(this.client);
    this.rules = new HestiaRules(this.client);
    this.logs = new HestiaLogs(
      this.client,
      config.baseUrl,
      apiPrefix,
    );
    this.collections = new HestiaCollections(this.client);
    this.blobs = new HestiaBlobClient(this.client, apiPrefix);
    this.capabilities = new HestiaCapabilities(this.client)
  }

  onAuthStateChange(callback: () => void) {
    this.onAuthStateChanged = callback;
  }

  async authenticated(): Promise<boolean> {
    if (this.tokenProvider.identity() === null) return false;
    try {
      await this.client.dispatch("system:core:heartbeat", { notifyAuthStateChange: false });
      return true;
    } catch {
      return false;
    }
  }

  private heartbeatTimer?: ReturnType<typeof setTimeout>;
  private defaultHeartbeatInterval = 5 * 60 * 1000;

  startHeartbeat(intervalMs?: number): void {
    this.stopHeartbeat();
    const ms = intervalMs ?? this.defaultHeartbeatInterval;
    const tick = () => {
      this.heartbeatTimer = setTimeout(async () => {
        await this.#heartbeat();
        tick();
      }, ms);
    };
    tick();
  }

  stopHeartbeat(): void {
    if (this.heartbeatTimer !== undefined) {
      clearTimeout(this.heartbeatTimer);
      this.heartbeatTimer = undefined;
    }
  }

  async #heartbeat(): Promise<void> {
    try {
      await this.client.dispatch("system:core:heartbeat");
    } catch {
      // dispatch already handled onAuthStateChanged
    }
  }

  collection<T extends Record<string, any>>(name: string) {
    return this.collections.documents<T>(name);
  }

  ready(){
      return this.client.ready()
  }
}
