import type { SimplePersistence } from "@asaidimu/utils-persistence";
import { ReactiveDataStore } from "@asaidimu/utils-store";
import { HestiaAuth } from "./auth/store";
import { HestiaCollections } from "./collections/store";
import {
    HestiaNetworkClient,
    type IdentityProvider,
} from "./core/client";
import { HestiaKeyStore } from "./system/api-keys/store";
import { HestiaUsers } from "./system/identity/store";
import type { UserIdentity } from "./system/identity/types";
import { HestiaLogs } from "./system/logs/store";
import { HestiaPolicies } from "./system/policies/store";
import { HestiaBlobClient } from "./blobs/store";
import { HestiaCapabilities } from "./system/capabilities/store";

export interface HestiaConfig {
  baseUrl: string;
  apiPrefix?: string;
  persistence?: SimplePersistence<AuthState>;
}

interface AuthState {
  identity: UserIdentity | null;
}

export class HestiaClient {
  readonly store: ReactiveDataStore<AuthState>;
  readonly client: HestiaNetworkClient;
  readonly auth: HestiaAuth;
  readonly users: HestiaUsers;
  readonly keys: HestiaKeyStore;
  readonly policies: HestiaPolicies;
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

    this.tokenProvider = tokenProvider;
    this.client = new HestiaNetworkClient(config.baseUrl, apiPrefix, tokenProvider, () =>
      this.onAuthStateChanged?.(),
    );

    this.auth = new HestiaAuth(this.client, tokenProvider);
    this.users = new HestiaUsers(this.client);
    this.keys = new HestiaKeyStore(this.client,);
    this.policies = new HestiaPolicies(this.client);
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

  authenticated(): boolean {
    return this.tokenProvider.identity() !== null;
  }

  collection<T extends Record<string, any>>(name: string) {
    return this.collections.documents<T>(name);
  }
}
