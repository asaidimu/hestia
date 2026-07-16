export interface PolicyOperation {
  name: string
  ruleKey: string
  description?: string
  intentType?: "COMMAND" | "QUERY"
}

export interface UpsertOperationRequest {
  ruleKey: string
  description?: string
  intentType?: "COMMAND" | "QUERY"
}

export interface PolicyRule {
  name: string
  ruleType?: string
  expression?: string
  description?: string
}

export interface UpsertRuleRequest {
  expression: string
  ruleType?: string
  description?: string
}

export interface ValidateRuleRequest {
  expression: string
  identity?: Record<string, unknown>
  resource?: Record<string, unknown>
  environment?: Record<string, unknown>
}

export interface ValidateRuleResult {
  valid: boolean
  compile?: string
  result?: boolean
}

export interface ReloadResult {
  operations: number
  rules: number
}
