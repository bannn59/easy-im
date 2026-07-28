export const LANG_STORAGE_KEY = 'easyim_lng';

export const SUPPORTED_LANGS = ['en', 'zh-CN'] as const;

export type AppLanguage = (typeof SUPPORTED_LANGS)[number];

export function isAppLanguage(value: string | null | undefined): value is AppLanguage {
  return value === 'en' || value === 'zh-CN';
}

function navigatorPrefersZh(): boolean {
  if (typeof navigator === 'undefined') return false;
  const candidates: string[] = [];
  if (Array.isArray(navigator.languages)) {
    candidates.push(...navigator.languages);
  }
  if (navigator.language) {
    candidates.push(navigator.language);
  }
  return candidates.some((lang) => lang.toLowerCase().startsWith('zh'));
}

/** Resolve initial UI language from localStorage, then browser, else English. */
export function resolveLanguage(): AppLanguage {
  if (typeof localStorage !== 'undefined') {
    const stored = localStorage.getItem(LANG_STORAGE_KEY);
    if (isAppLanguage(stored)) return stored;
  }
  if (navigatorPrefersZh()) return 'zh-CN';
  return 'en';
}
