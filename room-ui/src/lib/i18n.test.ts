import { beforeEach, describe, expect, it } from 'vitest'
import { dictionaries, errorText, locales, mapLocale, messageRows, resolveLocale, setLocale, t, translate, useI18n } from './i18n'

beforeEach(() => { localStorage.clear(); setLocale('en'); localStorage.clear() })
describe('locale resolution', () => {
  it.each(locales)('prefers nmr:locale for %s', code => {
    localStorage.setItem('nmr:locale', code)
    localStorage.setItem('i18nextLng', 'ja')
    expect(resolveLocale(localStorage, { language: 'de-DE', languages: ['fr'] })).toBe(code)
  })
  it('uses Navidrome locale keys before browser languages', () => {
    localStorage.setItem('i18nextLng', 'zh-Hant')
    localStorage.setItem('locale', 'es-MX')
    expect(resolveLocale(localStorage, { language: 'de', languages: ['fr'] })).toBe('zh-TW')
    localStorage.removeItem('i18nextLng')
    expect(resolveLocale(localStorage)).toBe('es')
  })
  it.each([['en-US', 'en'], ['zh-HK', 'zh-TW'], ['zh-TW', 'zh-TW'], ['zh-Hant-HK', 'zh-TW'], ['zh-SG', 'zh-CN'], ['zh', 'zh-CN'], ['ja-JP', 'ja'], ['es-AR', 'es'], ['fr-CA', 'fr'], ['de-AT', 'de']])('maps %s to %s', (language, expected) => {
    expect(resolveLocale(localStorage, { language, languages: [] })).toBe(expected)
    expect(mapLocale(language)).toBe(expected)
  })
  it('uses the first supported browser language and falls back to English', () => {
    expect(resolveLocale(localStorage, { languages: ['ar', 'fr-FR', 'de'], language: 'en' })).toBe('fr')
    expect(resolveLocale(localStorage, { languages: ['ar'], language: 'xx' })).toBe('en')
    expect(resolveLocale({ getItem() { throw new Error('blocked') } }, { languages: [], language: 'ja' })).toBe('ja')
  })
})
describe('localized messages', () => {
  it('reacts to selection, persists the shared key and substitutes parameters literally', () => {
    const { locale } = useI18n()
    setLocale('zh-CN')
    expect(locale.value).toBe('zh-CN')
    expect(localStorage.getItem('nmr:locale')).toBe('zh-CN')
    expect(t('onlineCount', { count: 12 })).toBe('12 人在线')
    expect(t('queued', { title: '$& <Song>' })).toBe('已点播 $& <Song>')
    expect(translate('fr', 'missing.key')).toBe('missing.key')
  })
  it.each(locales)('translates error codes in %s', code => {
    setLocale(code)
    for (const error of ['room_closed', 'membership_required', 'library_access_required', 'personal_queue_limit', 'room_manager_required', 'skip_permission_required', 'revision_conflict', 'navidrome_login_failed']) {
      expect(errorText({ code: error })).toBe(dictionaries[code][`error.${error}`])
    }
    expect(errorText('session_expired')).toBe(t('error.session_required'))
    expect(errorText({ code: 'unknown', message: 'raw server message' })).toBe(t('error.generic'))
  })
  it('has a complete dictionary and identical parameter names for every locale', () => {
    expect(new Set(messageRows.map(row => row[0])).size).toBe(messageRows.length)
    for (const row of messageRows) {
      expect(row.length, row[0]).toBe(8)
      const placeholders = (value: string) => [...value.matchAll(/\{\w+\}/g)].map(match => match[0]).sort()
      for (const value of row.slice(1)) {
        expect(value.trim(), row[0]).not.toBe('')
        expect(placeholders(value), row[0]).toEqual(placeholders(row[1]))
      }
    }
    for (const code of locales) expect(Object.keys(dictionaries[code])).toEqual(Object.keys(dictionaries.en))
  })
})
