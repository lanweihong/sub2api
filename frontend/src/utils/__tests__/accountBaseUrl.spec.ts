import { describe, expect, it } from 'vitest'

import { getDefaultApiKeyBaseUrl, isAnthropicCompatPlatform } from '../accountBaseUrl'

describe('accountBaseUrl', () => {
  it('returns provider-aligned defaults for anthropic compat platforms', () => {
    expect(getDefaultApiKeyBaseUrl('anthropic-zhipu')).toBe('https://open.bigmodel.cn/api/anthropic')
    expect(getDefaultApiKeyBaseUrl('anthropic-kimi')).toBe('https://api.moonshot.cn')
    expect(getDefaultApiKeyBaseUrl('anthropic-minimax')).toBe('https://api.minimax.chat')
    expect(getDefaultApiKeyBaseUrl('anthropic-qwen')).toBe('https://dashscope.aliyuncs.com/compatible-mode')
    expect(getDefaultApiKeyBaseUrl('anthropic-mimo')).toBe('https://api.mimo.xiaomi.com')
  })

  it('detects anthropic compat platforms by prefix', () => {
    expect(isAnthropicCompatPlatform('anthropic-zhipu')).toBe(true)
    expect(isAnthropicCompatPlatform('anthropic')).toBe(false)
    expect(isAnthropicCompatPlatform('openai')).toBe(false)
  })

  it('keeps existing defaults for first-party platforms', () => {
    expect(getDefaultApiKeyBaseUrl('openai')).toBe('https://api.openai.com')
    expect(getDefaultApiKeyBaseUrl('gemini')).toBe('https://generativelanguage.googleapis.com')
    expect(getDefaultApiKeyBaseUrl('antigravity')).toBe('https://cloudcode-pa.googleapis.com')
    expect(getDefaultApiKeyBaseUrl('anthropic')).toBe('https://api.anthropic.com')
  })
})
