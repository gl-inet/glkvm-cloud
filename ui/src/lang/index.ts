/*
 * @Author: LPY
 * @Date: 2025-05-29 18:14:55
 * @LastEditors: LPY
 * @LastEditTime: 2025-06-13 12:07:06
 * @FilePath: /kvm-cloud-frontend/src/lang/index.ts
 * @Description: 语言配置文件
 */
import { createI18n } from 'vue-i18n'
import zh from './locales/zh.json' 
import en from './locales/en.json'
import ja from './locales/ja.json'
import ko from './locales/ko.json'
import de from './locales/de.json'
import fr from './locales/fr.json'
import es from './locales/es.json'
import useLanguage from '@/hooks/useLanguage'
import { Languages } from 'gl-web-main'
import { AppLanguages } from '@/models/setting'

const i18n = createI18n({
    legacy: false,
    locale: Languages.EN, // 设置默认语言
    fallbackWarn: false, // 关闭控制台警告
    missingWarn: false, // 关闭控制台警告
    warnHtmlMessage: false, // 禁用HTML警告
    silentTranslationWarn:true,
    silentFallbackWarn: true,
})

/** 初始化语言 */
const initializeAllLanguage = () => {
    i18n.global.setLocaleMessage(Languages.ZH, zh)
    i18n.global.setLocaleMessage(Languages.EN, en)
    i18n.global.setLocaleMessage(AppLanguages.JA as string, ja)
    i18n.global.setLocaleMessage(AppLanguages.KO as string, ko)
    i18n.global.setLocaleMessage(AppLanguages.DE as string, de)
    i18n.global.setLocaleMessage(AppLanguages.FR as string, fr)
    i18n.global.setLocaleMessage(AppLanguages.ES as string, es)
    useLanguage().setLanguage()
}

export {
    initializeAllLanguage,
}

export default i18n