import { writable, derived } from "svelte/store";
import en from "./en";
import cn from "./cn";

const translations = { en, cn };

// 从本地存储获取初始语言，默认为英文
const initialLocale = localStorage.getItem("locale") || "en";

export const locale = writable(initialLocale);

// 监听语言变化并保存到本地存储
locale.subscribe((value) => {
    localStorage.setItem("locale", value);
});

// 核心翻译函数
export const t = derived(locale, ($locale) => (key, vars = {}) => {
    let text = translations[$locale][key] || key;

    // 处理变量替换，例如 {name}
    Object.keys(vars).forEach((v) => {
        text = text.replace(new RegExp(`{${v}}`, "g"), vars[v]);
    });

    return text;
});
