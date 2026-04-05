import type { AccountPlatform } from '@/types'

const anthropicCompatDefaultBaseUrls: Partial<Record<AccountPlatform, string>> = {
  'anthropic-compatible': '',
  'anthropic-zhipu': 'https://open.bigmodel.cn/api/anthropic',
  'anthropic-kimi': 'https://api.moonshot.cn',
  'anthropic-minimax': 'https://api.minimax.chat',
  'anthropic-qwen': 'https://dashscope.aliyuncs.com/compatible-mode',
  'anthropic-mimo': 'https://api.mimo.xiaomi.com',
}

export function isAnthropicCompatPlatform(platform?: string | null): platform is AccountPlatform {
  return typeof platform === 'string' && (platform === 'anthropic-compatible' || platform.startsWith('anthropic-'))
}

export function requiresExplicitApiKeyBaseUrl(platform?: AccountPlatform | string | null): boolean {
  return !!platform && isAnthropicCompatPlatform(platform) && getDefaultApiKeyBaseUrl(platform) === ''
}

export function getDefaultApiKeyBaseUrl(platform?: AccountPlatform | string | null): string {
  if (platform === 'openai' || platform === 'sora') return 'https://api.openai.com'
  if (platform === 'gemini') return 'https://generativelanguage.googleapis.com'
  if (platform === 'antigravity') return 'https://cloudcode-pa.googleapis.com'
  if (platform && isAnthropicCompatPlatform(platform)) {
    return anthropicCompatDefaultBaseUrls[platform] ?? ''
  }
  return 'https://api.anthropic.com'
}
