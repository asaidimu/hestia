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
  rules?: Record<string, unknown>
  description?: string
}

export interface UpdateRuleRequest {
  ruleType?: string
  syntax?: string
  expression?: string
  rules?: Record<string, unknown>
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
