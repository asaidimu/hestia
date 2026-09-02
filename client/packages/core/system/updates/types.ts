export interface UpdateStatus {
  version: string
  staged_version: string
  prepared: boolean
  last_check: number
}

export interface UpdateChangelog {
  version: string
  asset_name: string
  changelog: string
}

export interface UpdateCheckResult {
  checked: boolean
  staged: boolean
  version: string
  auto_apply: boolean
}

export interface UpdateAvailability {
  available: boolean
  version: string
}

export interface UpdateStageResult {
  staged: boolean
  version: string
}

export interface UpdateApplyResult {
  message: string
}
