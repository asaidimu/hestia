import type { Document } from "../../core/types"
import { type Transport } from "../../core/client"
import type { Notification, UnreadCount } from "./types"

export class HestiaNotificationStore {
  constructor(private client: Transport) {}

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
}
