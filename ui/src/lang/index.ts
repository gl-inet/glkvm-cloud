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
import useLanguage from '@/hooks/useLanguage'
import { Languages } from 'gl-web-main'

const i18n = createI18n({
    legacy: false,
    locale: Languages.EN, // 设置默认语言
    fallbackLocale: Languages.EN, // 缺的文案用英文
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
    useLanguage().setLanguage()
}

export {
    initializeAllLanguage,
}

export default i18n