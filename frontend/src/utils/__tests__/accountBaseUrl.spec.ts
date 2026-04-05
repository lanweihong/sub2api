import { describe, expect, it } from 'vitest'

import { getDefaultApiKeyBaseUrl, isAnthropicCompatPlatform, requiresExplicitApiKeyBaseUrl } from '../accountBaseUrl'

describe('accountBaseUrl', () => {
  it('returns provider-aligned defaults for anthropic compat platforms', () => {
    expect(getDefaultApiKeyBaseUrl('anthropic-compatible')).toBe('')
    expect(getDefaultApiKeyBaseUrl('anthropic-zhipu')).toBe('https://open.bigmodel.cn/api/anthropic')
    expect(getDefaultApiKeyBaseUrl('anthropic-kimi')).toBe('https://api.moonshot.cn')
    expect(getDefaultApiKeyBaseUrl('anthropic-minimax')).toBe('https://api.minimax.chat')
    expect(getDefaultApiKeyBaseUrl('anthropic-qwen')).toBe('https://dashscope.aliyuncs.com/compatible-mode')
    expect(getDefaultApiKeyBaseUrl('anthropic-mimo')).toBe('https://api.mimo.xiaomi.com')
  })

  it('detects anthropic compat platforms by prefix', () => {
    expect(isAnthropicCompatPlatform('anthropic-compatible')).toBe(true)
    expect(isAnthropicCompatPlatform('anthropic-zhipu')).toBe(true)
    expect(isAnthropicCompatPlatform('anthropic')).toBe(false)
    expect(isAnthropicCompatPlatform('openai')).toBe(false)
  })

  it('marks generic compat platform as requiring an explicit base url', () => {
    expect(requiresExplicitApiKeyBaseUrl('anthropic-compatible')).toBe(true)
    expect(requiresExplicitApiKeyBaseUrl('anthropic-zhipu')).toBe(false)
    expect(requiresExplicitApiKeyBaseUrl('anthropic')).toBe(false)
  })

  it('keeps existing defaults for first-party platforms', () => {
    expect(getDefaultApiKeyBaseUrl('openai')).toBe('https://api.openai.com')
    expect(getDefaultApiKeyBaseUrl('gemini')).toBe('https://generativelanguage.googleapis.com')
    expect(getDefaultApiKeyBaseUrl('antigravity')).toBe('https://cloudcode-pa.googleapis.com')
    expect(getDefaultApiKeyBaseUrl('anthropic')).toBe('https://api.anthropic.com')
  })
})
