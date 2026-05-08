import { createI18n } from 'vue-i18n'

// Catalogs live in /internal/i18n/catalogs/ as the single source of
// truth shared with the Go side. We ship all 5 supported locales
// statically — the catalogs are small (~50KB raw, ~10KB gz) and
// bundling them up front keeps theme/locale switching network-free,
// matching the SPA-first goal.
import itUI from '@catalogs/ui.it.json'
import enUI from '@catalogs/ui.en.json'
import esUI from '@catalogs/ui.es.json'
import frUI from '@catalogs/ui.fr.json'
import deUI from '@catalogs/ui.de.json'

export const i18n = createI18n({
  legacy: false,
  // Initial locale matches the API/CLI default (`en`); the UI store
  // calls hydrate() in App.vue setup which fetches the operator's
  // configured i18n.default_lang and overrides this on first boot.
  locale: 'en',
  fallbackLocale: 'en',
  messages: {
    it: itUI,
    en: enUI,
    es: esUI,
    fr: frUI,
    de: deUI,
  },
})
