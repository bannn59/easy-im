import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import en from './locales/en.json';
import zhCN from './locales/zh-CN.json';
import {
  isAppLanguage,
  LANG_STORAGE_KEY,
  resolveLanguage,
  SUPPORTED_LANGS,
  type AppLanguage,
} from './resolveLanguage';

export { SUPPORTED_LANGS, type AppLanguage };
export { LANG_STORAGE_KEY, resolveLanguage, isAppLanguage };

function setDocumentLang(lng: string) {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = lng;
  }
}

const initialLng = resolveLanguage();

void i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    'zh-CN': { translation: zhCN },
  },
  lng: initialLng,
  fallbackLng: 'en',
  interpolation: {
    escapeValue: false,
  },
});

// Use the resolved preference, not i18n.language, so document.lang is correct
// even if init is still settling.
setDocumentLang(initialLng);

/** Persist language preference and update document lang. */
export async function setAppLanguage(lng: AppLanguage): Promise<void> {
  await i18n.changeLanguage(lng);
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(LANG_STORAGE_KEY, lng);
  }
  setDocumentLang(lng);
}

export default i18n;
