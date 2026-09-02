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

export interface PolicyRule {
  id: string
  name: string
  ruleType?: string
  syntax?: string
  expression?: string
  rules?: RuleNode
  description?: string
  protected?: boolean
}

export interface CreateRuleRequest {
  ruleType?: string
  syntax?: string
  expression?: string
  rules?: RuleNode
  description?: string
}

export interface UpdateRuleRequest {
  ruleType?: string
  syntax?: string
  expression?: string
  rules?: RuleNode
  description?: string
  protected?: boolean
}

export interface RuleNode {
  type?: string
  name?: string
  expression?: string
  operator?: string
  conditions?: RuleNode[]
}

export interface ValidateRuleRequest {
  rule: string | RuleNode
  context?: {
    identity?: Record<string, unknown>
    resource?: Record<string, unknown>
    environment?: Record<string, unknown>
  }
}

export interface ValidateRuleResult {
  valid: boolean
  result?: boolean
  error?: string
}

export interface ReloadResult {
  operations: number
  rules: number
}
