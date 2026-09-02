export interface LogEntry {
  level: string
  ts: number
  caller: string
  msg: string
  fields?: Record<string, unknown>
  /** All other top-level keys: operation, duration, request_id, client_ip, user_id, email, error, stacktrace, etc. */
  extra?: Record<string, unknown>
}

export interface LogQuery {
  level?: string
  from?: string
  to?: string
  search?: string
  limit?: number
  offset?: number
}

export interface LogListResult {
  entries: LogEntry[]
  total: number
  has_more: boolean
}

// Fields commonly added by the access-log middleware
export interface AccessLogFields {
  method?: string
  path?: string
  operation?: string
  duration?: string
  request_id?: string
  client_ip?: string
  user_agent?: string
  user_id?: string
  email?: string
  tenant_id?: string
  status?: number
  code?: string
}
