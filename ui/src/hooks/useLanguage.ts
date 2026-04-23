/*
 * @Author: LPY
 * @Date: 2025-05-29 18:41:20
 * @LastEditors: LPY
 * @LastEditTime: 2025-09-04 17:01:57
 * @FilePath: \glkvm-cloud\ui\src\hooks\useLanguage.ts
 * @Description: 语言hook
 */

import i18n from '@/lang'
import { languageLabelMap, AppLanguages } from '@/models/setting'
import { computed } from 'vue'
import { LocalStorageKeys, useLocalStorage } from './useLocalStorage'
import { Languages } from 'gl-web-main'
/** 语言hook */
export default function useLanguage () {
    // @ts-ignore
    const browserLanguage = navigator.language || navigator.userLanguage || navigator.browserLanguage || ''
    const langPrefix = browserLanguage.split('-')[0].toLowerCase()

    // 根据浏览器语言推断默认语言
    const allLangs = Object.values(AppLanguages) as string[]
    const defaultLang = allLangs.includes(langPrefix) ? langPrefix as Languages : Languages.EN

    const {getValue, setValue} = useLocalStorage(LocalStorageKeys.STORAGE_LANGUAGE_KEY, defaultLang)

    const currentLang = computed(() => i18n.global.locale.value as Languages)

    const setLanguage = (lang = getValue()) => {
        i18n.global.locale.value = lang
        setValue(lang)
    }

    const isZh = computed(() => currentLang.value === Languages.ZH)

    const currentLangLabel = computed(() => languageLabelMap.get(currentLang.value))

    return {currentLang, setLanguage, isZh, t: i18n.global.t, currentLangLabel}
}

export const t = ((...args: any[]) =>
    (i18n.global.t as any)(...args)) as typeof i18n.global.t