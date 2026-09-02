export interface NotificationAction {
  label: string
  message?: string
  arguments?: Record<string, string>
  url?: string
}

export interface Notification {
  _id_: string
  user_id: string
  type: string
  subject: string
  body?: string
  data?: Record<string, unknown>
  actions?: NotificationAction[]
  read: boolean
  created_at: number
  _metadata_: Record<string, unknown>
}

export interface UnreadCount {
  count: number
}

export interface CreateNotificationInput {
  user_id: string
  subject: string
  type?: string
  body?: string
  data?: Record<string, unknown>
  actions?: NotificationAction[]
  expires_at?: number
}
