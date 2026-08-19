import type { Document } from "../../core/types"
import { type Transport } from "../../core/client"
import type { Notification, NotificationAction, UnreadCount } from "./types"

export class HestiaNotificationStore {
  constructor(private client: Transport<string>) {}

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