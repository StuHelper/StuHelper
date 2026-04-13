import { ref } from 'vue';
import { enUSMessages } from './en-US';
import { zhCNMessages } from './zh-CN';
const STORAGE_KEY = 'stuhelper:uniappx:locale';
const DEFAULT_LOCALE = 'zh-CN';
const messages = {
    'en-US': enUSMessages,
    'zh-CN': zhCNMessages,
};
export const activeLocale = ref(DEFAULT_LOCALE);
function getUniRuntime() {
    return globalThis.uni;
}
function normalizeLocale(value) {
    if (!value)
        return DEFAULT_LOCALE;
    return value.toLowerCase().startsWith('en') ? 'en-US' : 'zh-CN';
}
function readStoredLocale() {
    const runtime = getUniRuntime();
    if (!runtime)
        return null;
    try {
        const value = runtime.getStorageSync(STORAGE_KEY);
        return typeof value === 'string' && value ? normalizeLocale(value) : null;
    }
    catch {
        return null;
    }
}
function readSystemLocale() {
    const runtime = getUniRuntime();
    if (!runtime)
        return DEFAULT_LOCALE;
    try {
        if (typeof runtime.getLocale === 'function') {
            return normalizeLocale(runtime.getLocale());
        }
    }
    catch {
        // ignore system locale access failure
    }
    try {
        const system = runtime.getSystemInfoSync();
        return normalizeLocale(system.language);
    }
    catch {
        return DEFAULT_LOCALE;
    }
}
function formatMessage(template, params) {
    if (!params)
        return template;
    return template.replace(/\{(\w+)\}/g, (_match, key) => {
        const value = params[key];
        return value == null ? '' : String(value);
    });
}
export function getLocale() {
    return activeLocale.value;
}
export function setLocale(locale) {
    activeLocale.value = locale;
    const runtime = getUniRuntime();
    if (!runtime)
        return;
    try {
        runtime.setStorageSync(STORAGE_KEY, locale);
    }
    catch {
        // ignore storage failure
    }
    syncAppChrome();
}
export function bootstrapLocale() {
    const locale = readStoredLocale() ?? readSystemLocale();
    activeLocale.value = locale;
    syncAppChrome();
    return locale;
}
export function translate(key, params) {
    const message = messages[activeLocale.value][key] ?? messages[DEFAULT_LOCALE][key] ?? key;
    return formatMessage(message, params);
}
export function isEnglishLocale() {
    return activeLocale.value === 'en-US';
}
export function setPageTitle(key, params) {
    const runtime = getUniRuntime();
    if (!runtime)
        return;
    try {
        runtime.setNavigationBarTitle({ title: translate(key, params) });
    }
    catch {
        // ignore title update failure
    }
}
export function syncAppChrome() {
    const runtime = getUniRuntime();
    if (!runtime)
        return;
    const tabs = [
        translate('common.tabs.home'),
        translate('common.tabs.course'),
        translate('common.tabs.review'),
        translate('common.tabs.user'),
    ];
    tabs.forEach((text, index) => {
        try {
            runtime.setTabBarItem({ index, text });
        }
        catch {
            // ignore when tabbar is unavailable
        }
    });
}
