export interface Policy {
  id: string
  operationName: string
  ruleName: string
  enabled: boolean
  protected: boolean
}

export interface CreatePolicyRequest {
  ruleName: string
}

export interface UpdatePolicyRuleRequest {
  ruleName: string
}

export interface SetPolicyEnabledRequest {
  enabled: boolean
}
