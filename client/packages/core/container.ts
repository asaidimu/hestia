import type { SimplePersistence } from "@asaidimu/utils-persistence";
import { ReactiveDataStore } from "@asaidimu/utils-store";
import { HestiaAuth } from "./system/auth/store";
import { HestiaCollections } from "./system/collections/store";
import {
  HttpTransport,
  type Transport,
  type IdentityProvider,
} from "./core/client";
import { WailsTransport } from "./core/wails-transport";
import { HestiaKeyStore } from "./system/apikeys/store";
import { HestiaUsers } from "./system/users/store";
import type { UserIdentity } from "./system/users/types";
import { HestiaAuditLogs } from "./system/audit/store";
import { HestiaAppLogs } from "./system/logs/store";
import { HestiaPolicies } from "./system/policies/store";
import { HestiaRules } from "./system/policies/rules";
import { HestiaBlobClient } from "./system/blobs/store";
import type { UploadPersistence } from "./system/blobs/types";
import { HestiaCore } from "./system/core/store";
import { HestiaNotificationStore } from "./system/notifications/store";
import { HestiaScheduleStore } from "./system/schedules/store";
import { HestiaSettingStore } from "./system/settings/store";
import { HestiaUpdates } from "./system/updates/store";
import { HestiaWorkflowStore } from "./system/workflows/store";
import { HestiaOperations } from "./system/operations/store";

export interface HestiaConfig {
  baseUrl: string;
  apiPrefix?: string;
  apiKey?: string;
  persistence?: SimplePersistence<AuthState>;
  uploadPersistence?: UploadPersistence;
  transport?: Transport;
}

interface AuthState {
  identity: UserIdentity | null;
}

/*
 * @note #worb9o issue : Weak Authentication State persistence
 *
 * Opening two tabs on the same browser client causes the user to be
 * deauthenticated
 */

export class HestiaClient {
  readonly store: ReactiveDataStore<AuthState>;
  readonly client: Transport;
  readonly auth: HestiaAuth;
  readonly users: HestiaUsers;
  readonly keys: HestiaKeyStore;
  readonly policies: HestiaPolicies;
  readonly rules: HestiaRules;
  readonly auditLogs: HestiaAuditLogs;
  readonly appLogs: HestiaAppLogs;
  readonly collections: HestiaCollections;
  readonly blobs: HestiaBlobClient;
  readonly core: HestiaCore;
  readonly notifications: HestiaNotificationStore;
  readonly schedules: HestiaScheduleStore;
  readonly settings: HestiaSettingStore;
  readonly updates: HestiaUpdates;
  readonly workflows: HestiaWorkflowStore;
  readonly operations: HestiaOperations;
  private tokenProvider: IdentityProvider;
  private readonly apiKeyMode: boolean;

  private onAuthStateChanged = new Set<() => void>();

  constructor(config: HestiaConfig) {
    this.store = new ReactiveDataStore<AuthState>(
      { identity: null },
      config.persistence,
    );

    const tokenProvider: IdentityProvider = {
      identity: () => this.store.get().identity,
      setIdentity: async (identity) =>
        void (await this.store.set({ identity })),
      clear: async () => void (await this.store.set({ identity: null })),
    };

    const apiPrefix = config.apiPrefix ?? "/api";
    const apiKeyHeaders = config.apiKey
      ? { "X-Api-Key": config.apiKey }
      : undefined;

    const onUnauthorized = () => {
      tokenProvider.clear();
      this.onAuthStateChanged.forEach(i => i())
    };

    this.tokenProvider = tokenProvider;
    this.apiKeyMode = !!config.apiKey;
    if (config.transport instanceof WailsTransport) {
      config.transport.configure(config.baseUrl, apiPrefix, tokenProvider);
      config.transport.setOnUnauthorized(onUnauthorized);
      this.client = config.transport;
    } else {
      this.client =
        config.transport ??
        new HttpTransport(
          config.baseUrl,
          apiPrefix,
          onUnauthorized,
          apiKeyHeaders,
        );
    }

    this.auth = new HestiaAuth(this.client, tokenProvider);
    this.users = new HestiaUsers(this.client);
    this.keys = new HestiaKeyStore(this.client);
    this.policies = new HestiaPolicies(this.client);
    this.rules = new HestiaRules(this.client);
    this.auditLogs = new HestiaAuditLogs(
      this.client,
      config.baseUrl,
      apiPrefix,
    );
    this.appLogs = new HestiaAppLogs(this.client, config.baseUrl, apiPrefix);
    this.collections = new HestiaCollections(this.client);
    this.blobs = new HestiaBlobClient(
      this.client,
      apiPrefix,
      config.uploadPersistence,
    );
    this.core = new HestiaCore(this.client);
    this.notifications = new HestiaNotificationStore(this.client);
    this.schedules = new HestiaScheduleStore(this.client);
    this.settings = new HestiaSettingStore(this.client);
    this.updates = new HestiaUpdates(this.client);
    this.workflows = new HestiaWorkflowStore(this.client);
    this.operations = new HestiaOperations(this.client);
  }

  // @note #auth-20261029-007 todo P2 #client,#ergonomics : onAuthStateChange is a single callback slot that fires only on 401
  // @author claude
  //
  // It assigns rather than subscribes: a second call silently replaces the
  // first and nothing is returned to unsubscribe. It fires only when a request
  // gets 401, not on login, logout or restore, despite the name, so consumers
  // (the IAM wrapper) have to call back into their own provider to find out
  // what happened and the client store and IAM store can drift.
  // Proposal: onAuthStateChange(cb: (identity: UserIdentity | null) => void):
  // () => void, backed by store.watch("identity"); keep 401 -> clear.

  // @note #auth-20261029-009 todo P3 #client,#auth,#ergonomics : No cross-tab auth signalling
  // @author claude
  // @see #auth-20261029-007
  //
  // Logout in tab A revokes the session id (blocklist) but tab B only learns at
  // its next heartbeat (up to 5 min) or 401. Login in tab A is never seen by
  // tab B. Proposal: post "changed" on a BroadcastChannel (name scoped to
  // baseUrl) after login/logout and, on receipt, re-run auth.session(). The
  // server stays the source of truth: never broadcast claims. Do not post from
  // the handler of a received message (ping-pong).

  onAuthStateChange(callback: () => void) {
    this.onAuthStateChanged.add(callback)
  }

  // @note #auth-20261029-006 issue P1 #client,#auth,#ergonomics : authenticated() returns false without asking the server when local identity is null
  // @author claude
  // @see #auth-20261029-003
  //
  // This is the direct cause of "every tab asks me to log in": a new tab has an
  // empty client store, so authenticated() answers false without a request even
  // though the HttpOnly cookie is valid. The name promises a server-confirmed
  // state but it only confirms when a local identity already exists.
  // Fix: once auth.session() exists (#auth-20261029-003) call it here
  // (null -> false, throw -> propagate), keeping the apiKeyMode shortcut. The
  // heartbeat is a poor probe because it carries no claims.

  async authenticated(): Promise<boolean> {
    if (this.apiKeyMode) return true;
    if (this.tokenProvider.identity() === null) return false;
    try {
      await this.client.dispatch("system:core:heartbeat", {
        notifyAuthStateChange: false,
      });
      return true;
    } catch {
      return false;
    }
  }

  private heartbeatTimer?: ReturnType<typeof setTimeout>;
  private defaultHeartbeatInterval = 5 * 60 * 1000;

  // @note #auth-20261029-008 todo P2 #client,#auth,#ergonomics : Heartbeat lifecycle is hand-managed by every consumer, and it is what keeps the session alive
  // @author claude
  // @see #auth-20261029-001
  //
  // Server session = 8h absolute (DefaultSessionTTL) with a 30min idle limit
  // (DefaultIdleTTL); any authenticated request after 15min (DefaultRefreshTTL)
  // re-issues the cookie, and NoRefreshCommands is never populated, so this
  // 5min heartbeat is what makes access uninterrupted while a tab is open.
  // Consumers must start/stop it by hand (the IAM wrapper does so only inside
  // onAuthenticate/onDeauthenticate), so a restored session, an expiry-driven
  // logout or a cross-tab logout never starts/stops it.
  // Proposal: HestiaClient owns it, starting when store.identity becomes
  // non-null and stopping on null; keep start/stopHeartbeat as an opt-out.
  // Every tab runs its own timer (N tabs, N heartbeats): harmless, but could be
  // leader-elected. A suspended machine can exceed the idle limit; a re-login
  // there is expected behaviour, not a bug.

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

  view<T extends Record<string, any>>(name: string) {
    return this.collections.view<T>(name);
  }

  ready() {
    return this.client.ready();
  }
}
