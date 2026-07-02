/**
 * Channel monitor shared constants.
 *
 * Single source of truth for provider/status string values used by both the
 * admin (`views/admin/ChannelMonitorView.vue`) and user-facing
 * (`views/user/ChannelStatusView.vue`) screens, plus the shared composable
 * `useChannelMonitorFormat`.
 */

import type { APIMode, Provider, MonitorStatus } from '@/api/admin/channelMonitor'

export const PROVIDER_OPENAI: Provider = 'openai'
export const PROVIDER_ANTHROPIC: Provider = 'anthropic'
export const PROVIDER_GEMINI: Provider = 'gemini'
export const PROVIDER_ANTHROPIC_COMPATIBLE: Provider = 'anthropic-compatible'
export const PROVIDER_ANTHROPIC_ZHIPU: Provider = 'anthropic-zhipu'
export const PROVIDER_ANTHROPIC_KIMI: Provider = 'anthropic-kimi'
export const PROVIDER_ANTHROPIC_MINIMAX: Provider = 'anthropic-minimax'
export const PROVIDER_ANTHROPIC_QWEN: Provider = 'anthropic-qwen'
export const PROVIDER_ANTHROPIC_MIMO: Provider = 'anthropic-mimo'

export const ANTHROPIC_COMPAT_PROVIDERS: readonly Provider[] = [
  PROVIDER_ANTHROPIC_COMPATIBLE,
  PROVIDER_ANTHROPIC_ZHIPU,
  PROVIDER_ANTHROPIC_KIMI,
  PROVIDER_ANTHROPIC_MINIMAX,
  PROVIDER_ANTHROPIC_QWEN,
  PROVIDER_ANTHROPIC_MIMO,
]

export const API_MODE_CHAT_COMPLETIONS: APIMode = 'chat_completions'
export const API_MODE_RESPONSES: APIMode = 'responses'

export const PROVIDERS: readonly Provider[] = [
  PROVIDER_ANTHROPIC,
  PROVIDER_OPENAI,
  PROVIDER_GEMINI,
  ...ANTHROPIC_COMPAT_PROVIDERS,
]

export const API_MODES: readonly APIMode[] = [
  API_MODE_CHAT_COMPLETIONS,
  API_MODE_RESPONSES,
]

export const STATUS_OPERATIONAL: MonitorStatus = 'operational'
export const STATUS_DEGRADED: MonitorStatus = 'degraded'
export const STATUS_FAILED: MonitorStatus = 'failed'
export const STATUS_ERROR: MonitorStatus = 'error'

export const MONITOR_STATUSES: readonly MonitorStatus[] = [
  STATUS_OPERATIONAL,
  STATUS_DEGRADED,
  STATUS_FAILED,
  STATUS_ERROR,
]

/** Default polling interval (seconds) for new monitors. */
export const DEFAULT_INTERVAL_SECONDS = 60
