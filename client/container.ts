import { ArtifactContainer } from "@asaidimu/utils-artifacts";
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
import { UserIdentity } from "./system/identity/types";
import { HestiaLogs } from "./system/logs/store";
import { HestiaPolicies } from "./system/policies/store";
import { HestiaBlobClient } from "./blobs/store";
import { HestiaCapabilities } from "./system/capabilities/store";

export interface HestiaConfig {
  baseUrl: string;
  persistence?: SimplePersistence<AuthState>;
}

interface AuthState {
  access: string | null;
  refresh: string | null;
  identity: UserIdentity | null;
}

type Registry = Record<string, any>;

export class HestiaClient {
  readonly store: ReactiveDataStore<AuthState>;
  readonly container: ArtifactContainer<Registry, any>;
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
      { access: null, refresh: null, identity: null },
      config.persistence,
    );

    this.container = new ArtifactContainer<Registry, any>(this.store);

    const tokenProvider: IdentityProvider = {
      identity: () => this.store.get().identity,
      token: (key: "access" | "refresh") => this.store.get()[key],
      setTokens: async (access: string, refresh: string) =>
        void (await this.store.set({ access, refresh })),
      setIdentity: async (identity) =>
        void (await this.store.set({ identity })),
      clear: async () =>
        void (await this.store.set({ access: null, refresh: null })),
    };

    this.tokenProvider = tokenProvider;
    this.client = new HestiaNetworkClient(config.baseUrl, tokenProvider, () =>
      this.onAuthStateChanged?.(),
    );

    this.auth = new HestiaAuth(this.client, tokenProvider);
    this.users = new HestiaUsers(this.client);
    this.keys = new HestiaKeyStore(this.client,);
    this.policies = new HestiaPolicies(this.client);
    this.logs = new HestiaLogs(
      this.client,
      config.baseUrl,
    );
    this.collections = new HestiaCollections(this.client);
    this.blobs = new HestiaBlobClient(this.client);
    this.capabilities = new HestiaCapabilities(this.client)
  }

  onAuthStateChange(callback: () => void) {
    this.onAuthStateChanged = callback;
  }

  authenticated(): boolean {
    return this.tokenProvider.token("access") !== null;
  }

  collection<T extends Record<string, any>>(name: string) {
    return this.collections.documents<T>(name);
  }
}
