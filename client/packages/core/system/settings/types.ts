export interface SettingDocument {
  _id_: string
  key: string
  value: Record<string, unknown>
  _metadata_: Record<string, unknown>
}
