import { ref } from 'vue';
import en from '../../../i18n/locales/en.json';
export function useI18n() {
  return { locale: ref('en'), t: (key: string, params: Record<string, unknown> = {}) => {
    const value = key.split('.').reduce((result: any, part) => result?.[part], en) ?? key;
    return typeof value === 'string' ? value.replace(/\{(\w+)\}/g, (match, name) => params[name] === undefined ? match : String(params[name])) : value;
  } };
}
