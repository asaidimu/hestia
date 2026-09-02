/**
 * In-memory pub/sub event bus.
 *
 * The real server pushes Server-Sent Events over HTTP; the mock pushes the
 * same payloads through this bus. `openStream()` subscribes to a topic and
 * renders SSE-framed messages, so consuming code (notifications, audit
 * streams, workflow run streams) behaves identically.
 */

export type BusListener = (payload: unknown) => void;

export class EventBus {
  private topics = new Map<string, Set<BusListener>>();

  subscribe(topic: string, listener: BusListener): () => void {
    let set = this.topics.get(topic);
    if (!set) {
      set = new Set();
      this.topics.set(topic, set);
    }
    set.add(listener);
    return () => {
      set!.delete(listener);
      if (set!.size === 0) this.topics.delete(topic);
    };
  }

  emit(topic: string, payload: unknown): void {
    const set = this.topics.get(topic);
    if (!set || set.size === 0) return;
    for (const listener of [...set]) {
      try {
        listener(payload);
      } catch {
        // A faulty subscriber must not break the bus.
      }
    }
  }

  listenerCount(topic?: string): number {
    if (topic) return this.topics.get(topic)?.size ?? 0;
    let total = 0;
    for (const set of this.topics.values()) total += set.size;
    return total;
  }

  clear(): void {
    this.topics.clear();
  }
}

/** Canonical topic names used by the mock server. */
export const Topics = {
  notifications: (userId: string) => `notifications:${userId}`,
  notificationsAll: "notifications:all",
  audit: "audit",
  logs: (level?: string) => (level ? `logs:${level}` : "logs"),
  workflowRun: (runId: string) => `workflow:run:${runId}`,
  collections: (name: string) => `collection:${name}`,
} as const;
