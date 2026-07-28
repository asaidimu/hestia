export interface Notification {
  _id_: string
  user_id: string
  type: string
  subject: string
  body?: string
  data?: Record<string, unknown>
  read: boolean
  created_at: number
  _metadata_: Record<string, unknown>
}

export interface UnreadCount {
  count: number
}
