export interface RateLimitConfig {
  enabled: boolean
  identity: string
  capacity: number
  refill: number
  period: number
}

export interface ThrottleAction {
  message?: string
  input?: Record<string, unknown>
}

export interface ThrottleConfig {
  limit: number
  window: number
  action?: ThrottleAction
}

export interface Policy {
  operation: string
  key: string
  rule: string
  enabled: boolean
  protected: boolean
  rateLimit?: RateLimitConfig
  throttle?: ThrottleConfig
}

export interface CreatePolicyRequest {
  rule: string
  rateLimit?: RateLimitConfig
  throttle?: ThrottleConfig
}

export interface UpdatePolicyRuleRequest {
  rule: string
}

export interface SetPolicyEnabledRequest {
  enabled: boolean
}

export interface UpdatePolicyRequest {
  rule?: string
  enabled?: boolean
  rateLimit?: RateLimitConfig | null
  throttle?: ThrottleConfig | null
}
