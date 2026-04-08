/*
 * @Author: LPY
 * @Date: 2025-05-30 09:24:47
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-09 16:43:06
 * @FilePath: \glkvm-cloud\ui\src\models\setting.ts
 * @Description: 设置相关类型声明
 */

import { Languages, SelectOptions } from 'gl-web-main'

/** 扩展语言代码（gl-web-main 的 Languages 枚举仅含 zh/en，此处补充其余语言） */
export const AppLanguages = {
    ...Languages,
    JA: 'ja' as unknown as Languages,
    KO: 'ko' as unknown as Languages,
    DE: 'de' as unknown as Languages,
    FR: 'fr' as unknown as Languages,
    ES: 'es' as unknown as Languages,
} as const

export type AppLanguage = (typeof AppLanguages)[keyof typeof AppLanguages]

/** 语言对应的label映射 */
export const languageLabelMap = new Map<Languages, string>([
    [Languages.ZH, '中文'],
    [Languages.EN, 'English'],
    [AppLanguages.JA, '日本語'],
    [AppLanguages.KO, '한국어'],
    [AppLanguages.DE, 'Deutsch'],
    [AppLanguages.FR, 'Français'],
    [AppLanguages.ES, 'Español'],
])

/** 语言选择options */
export const languageOptions = Object.values(AppLanguages).map(lang => new SelectOptions(lang, languageLabelMap.get(lang as Languages)))

export interface Theme {
    attribute: string,
    content: any,
}

/** 操作系统 */
export enum OperatingSystemEnum {
    GL_KVM = 'gl-kvm',
    LINUX = 'linux',
    WINDOWS = 'windows',
    MAC_OS = 'macOS',
}

/** 操作系统label映射 */
export const operatingSystemLabelMap = new Map<OperatingSystemEnum, string>([
    [OperatingSystemEnum.GL_KVM, 'GL-iNet KVM'],
    [OperatingSystemEnum.LINUX, 'Linux'],
    [OperatingSystemEnum.WINDOWS, 'Windows'],
    [OperatingSystemEnum.MAC_OS, 'MacOS'],
])