export interface LogEntry {
  level: string
  ts: number
  caller: string
  msg: string
  fields?: Record<string, unknown>
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
