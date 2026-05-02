import { writable, derived } from "svelte/store";
import en from "./en";
import cn from "./cn";

const translations = { en, cn };
const defaultLocale = "en";

function normalizeLocale(value) {
    return translations[value] ? value : defaultLocale;
}

// Read the saved locale when available; otherwise default to English.
const initialLocale = normalizeLocale(localStorage.getItem("locale"));

export const locale = writable(initialLocale);

locale.subscribe((value) => {
    localStorage.setItem("locale", normalizeLocale(value));
});

export const t = derived(locale, ($locale) => (key, vars = {}) => {
    let text = translations[normalizeLocale($locale)]?.[key] || key;

    Object.keys(vars).forEach((v) => {
        text = text.replace(new RegExp(`{${v}}`, "g"), vars[v]);
    });

    return text;
});
