import { describe, expect, it } from 'vitest'

import { platformBadgeLightClass, platformTextClass } from '../platformColors'

const compatPlatforms = [
  'anthropic-compatible',
  'anthropic-zhipu',
  'anthropic-kimi',
  'anthropic-minimax',
  'anthropic-qwen',
  'anthropic-mimo',
]

describe('platformColors anthropic-compatible palette', () => {
  it('reuses the anthropic palette for compat platforms', () => {
    const anthropicText = platformTextClass('anthropic')
    const anthropicBadge = platformBadgeLightClass('anthropic')

    for (const platform of compatPlatforms) {
      expect(platformTextClass(platform)).toBe(anthropicText)
      expect(platformBadgeLightClass(platform)).toBe(anthropicBadge)
    }
  })
})
