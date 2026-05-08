import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useUIStore } from '@/stores/ui'
import { i18n } from '@/locales'

describe('ui store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('defaults to catppuccin-mocha + empty locale (server-default fetched on hydrate)', () => {
    const ui = useUIStore()
    expect(ui.preset).toBe('catppuccin-mocha')
    // Empty sentinel — hydrate() fetches /api/v1/version on first
    // boot to pick the operator's i18n.default_lang. The user's
    // explicit choice via setLocale wins forever after.
    expect(ui.locale).toBe('')
  })

  it('setPreset writes data-preset on documentElement', () => {
    const ui = useUIStore()
    ui.setPreset('tokyo-night')
    expect(document.documentElement.dataset.preset).toBe('tokyo-night')
  })

  it('setPreset rejects invalid values silently', () => {
    const ui = useUIStore()
    ui.setPreset('not-a-preset' as never)
    expect(ui.preset).toBe('catppuccin-mocha')
  })

  it('setLocale flips i18n.global.locale and <html lang>', () => {
    const ui = useUIStore()
    ui.setLocale('en')
    expect(ui.locale).toBe('en')
    expect(document.documentElement.lang).toBe('en')
    expect((i18n.global.locale as unknown as { value: string }).value).toBe('en')
  })

  it('hydrate applies both preset and locale at once (avoids first-paint flash)', () => {
    const ui = useUIStore()
    ui.preset = 'solarized-light'
    ui.locale = 'fr'
    ui.hydrate()
    expect(document.documentElement.dataset.preset).toBe('solarized-light')
    expect(document.documentElement.lang).toBe('fr')
  })
})
