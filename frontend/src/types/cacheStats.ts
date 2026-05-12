export type CacheStatsDimension =
  | 'summary'
  | 'user'
  | 'api_key'
  | 'account'
  | 'group'
  | 'model'
  | 'endpoint'
  | 'day'
  | 'hour'

export type CacheStatsModelSource = 'requested' | 'upstream' | 'mapping'
export type CacheStatsEndpointSource = 'inbound' | 'upstream'

export interface CacheStatsItem {
  key: string
  label: string
  requests: number
  input_tokens: number
  output_tokens: number
  cache_creation_tokens: number
  cache_read_tokens: number
  total_tokens: number
  cache_token_rate: number
  cache_read_rate: number
  cache_write_rate: number
  cost: number
  actual_cost: number
  account_cost: number
}

export interface CacheStatsResponse {
  dimension: CacheStatsDimension
  model_source?: CacheStatsModelSource
  endpoint_source?: CacheStatsEndpointSource
  items: CacheStatsItem[]
  summary: CacheStatsItem
  start_date: string
  end_date: string
}
