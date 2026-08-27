import type { Document } from "../../core/types"
import { type Transport, type StreamHandlers, type StreamOptions } from "../../core/client"
import type {
  Notification,
  NotificationAction,
  UnreadCount,
  CreateNotificationInput,
} from "./types"

export class HestiaNotificationStore {
  constructor(private client: Transport<string>) {}

  /**
   * Create an in-app notification for a user (administrator-gated). Content
   * is taken verbatim — no template rendering.
   */
  async create(input: CreateNotificationInput): Promise<Document<Notification>> {
    const res = await this.client.dispatch<{
      data: Document<Notification>
    }>("system:notifications:notification:create", { payload: input })
    return res.data!.data
  }

  async list(): Promise<Document<Notification>[]> {
    const res = await this.client.dispatch<{ data: Document<Notification>[] }>(
      "system:notifications:notification:list",
    )
    return res.data?.data ?? []
  }

  async markRead(notificationId: string): Promise<void> {
    await this.client.dispatch("system:notifications:notification:read", {
      arguments: { notification_id: notificationId },
    })
  }

  async markAllRead(): Promise<void> {
    await this.client.dispatch("system:notifications:read:all")
  }

  async countUnread(): Promise<number> {
    const res = await this.client.dispatch<{ data: Document<UnreadCount> }>(
      "system:notifications:unread:count",
    )
    return res.data?.data?.count ?? 0
  }

  /**
   * Stream new notifications in real-time via SSE. The `onMessage` callback
   * receives each notification as a parsed `Document<Notification>` object.
   *
   * Returns an `AbortController` — call `abort()` to close the stream.
   *
   * @example
   * ```ts
   * const ctrl = store.stream({
   *   onMessage(doc) {
   *     console.log("new notification:", doc.subject)
   *   },
   *   onError(err) {
   *     console.error("stream error:", err)
   *   },
   * })
   * // later: ctrl.abort()
   * ```
   */
  stream(
    handlers: {
      onMessage: (notification: Document<Notification>) => void
      onError?: (err: Error) => void
      onOpen?: () => void
      onClose?: () => void
    },
    options?: StreamOptions,
  ): AbortController {
    const controller = new AbortController()
    const signal = AbortSignal.any([controller.signal, ...(options?.signal ? [options.signal] : [])])

    this.client
      .openStream(
        "/system/notifications/notification/stream",
        {
          onMessage(data) {
            try {
              const parsed = JSON.parse(data)
              const doc = parsed.data ?? parsed
              handlers.onMessage(doc)
            } catch (err) {
              handlers.onError?.(err instanceof Error ? err : new Error(String(err)))
            }
          },
          onError: handlers.onError,
          onOpen: handlers.onOpen,
          onClose: handlers.onClose,
        },
        { ...options, signal },
      )
      .catch((err) => {
        handlers.onError?.(err instanceof Error ? err : new Error(String(err)))
      })

    return controller
  }

  /**
   * Run a notification action: dispatch its message with the action's
   * arguments under the current identity. Authorization is enforced by the
   * policy engine like any other dispatch. Actions without a message (URL-only
   * actions) are link navigation, not dispatch — the caller opens `url`
   * itself.
   */
  async dispatchAction(action: NotificationAction): Promise<void> {
    if (!action.message) {
      throw new Error("action has no message to dispatch")
    }
    await this.client.dispatch(action.message, {
      arguments: action.arguments ?? {},
    })
  }
}