import { describe, expect, it } from 'vitest'

import { getPlatformTagClass } from '../types'

const compatPlatforms = [
  'anthropic-compatible',
  'anthropic-zhipu',
  'anthropic-kimi',
  'anthropic-minimax',
  'anthropic-qwen',
  'anthropic-mimo',
]

describe('channel pricing tag classes', () => {
  it('uses the anthropic tag palette for compat platforms', () => {
    const anthropicTagClass = getPlatformTagClass('anthropic')

    for (const platform of compatPlatforms) {
      expect(getPlatformTagClass(platform)).toBe(anthropicTagClass)
    }
  })
})
