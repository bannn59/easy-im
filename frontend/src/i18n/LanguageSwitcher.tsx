import { useTranslation } from 'react-i18next';
import { isAppLanguage, setAppLanguage, SUPPORTED_LANGS, type AppLanguage } from './index';

const LABELS: Record<AppLanguage, string> = {
  en: 'English',
  'zh-CN': '中文',
};

export function LanguageSwitcher() {
  const { t, i18n } = useTranslation();
  const current: AppLanguage = isAppLanguage(i18n.language) ? i18n.language : 'en';

  return (
    <div className="lang-switch" role="group" aria-label={t('language.switcherAria')}>
      {SUPPORTED_LANGS.map((lng) => {
        const pressed = current === lng;
        return (
          <button
            key={lng}
            type="button"
            lang={lng}
            className={pressed ? 'lang-switch__btn lang-switch__btn--active' : 'lang-switch__btn'}
            aria-pressed={pressed}
            onClick={() => {
              if (!pressed) void setAppLanguage(lng);
            }}
          >
            {LABELS[lng]}
          </button>
        );
      })}
    </div>
  );
}
