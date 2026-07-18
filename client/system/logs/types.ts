export interface AuditEntry {
  event_id: string
  occurred_at: string
  recorded_at: string
  trace_id?: string
  request_id?: string
  actor_id: string
  actor_type: "user" | "service" | "system" | "anonymous"
  on_behalf_of_id?: string
  auth_method?: "password" | "oauth" | "api_key" | "mtls" | "sso" | "service_account" | "none"
  session_id?: string
  operation: "create" | "read" | "update" | "delete" | "login" | "logout" | "grant" | "revoke" | "execute" | "other"
  resource_type: string
  resource_id?: string
  event_name: string
  status: "success" | "failure" | "denied" | "error"
  severity?: "info" | "warning" | "critical"
  error_code?: string
  error_message?: string
  latency_ms: number
  source_ip?: string
  user_agent?: string
  service_name: string
  region?: string
}

export type RequestLogEntry = AuditEntry

export interface LogFilter {
  actor_id?: string
  actor_type?: string
  operation?: string
  status?: string
  resource_type?: string
  trace_id?: string
  start?: string
  end?: string
}
