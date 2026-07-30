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
  id: string
  operationName: string
  ruleName: string
  enabled: boolean
  protected: boolean
  rateLimit?: RateLimitConfig
  throttle?: ThrottleConfig
}

export interface CreatePolicyRequest {
  ruleName: string
  rateLimit?: RateLimitConfig
  throttle?: ThrottleConfig
}

export interface UpdatePolicyRuleRequest {
  ruleName: string
}

export interface SetPolicyEnabledRequest {
  enabled: boolean
}

export interface UpdatePolicyRequest {
  ruleName?: string
  enabled?: boolean
  rateLimit?: RateLimitConfig | null
  throttle?: ThrottleConfig | null
}
