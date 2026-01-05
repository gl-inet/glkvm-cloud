/*
 * @Author: LPY
 * @Date: 2025-05-30 10:18:18
 * @LastEditors: LPY
 * @LastEditTime: 2026-01-05 14:22:12
 * @FilePath: \glkvm-cloud\ui\src\hooks\useLocalStorage.ts
 * @Description: 存储hook
 */
import { ref } from 'vue'

/** 整个系统 */
export enum LocalStorageKeys {
    /** 存储语言的key */
    STORAGE_LANGUAGE_KEY = 'language',
    /** 主题色 */
    THEME_MODE_KEY = 'gl-kvm-theme-mode-key',
    /** 两步验证所需信息 */
    TWO_FACTOR_INFO_KEY = 'two-factor-info',
    /** 侧边栏手动控制展开收缩状态 */
    SIDEBAR_MANUAL_CONTROL_KEY = 'sidebar-manual-control',
    /** 版本号 */
    VERSION = 'version',
}

/** 
 * @template T 需要使用的数据类型
 * @param key 需要使用哪种数据
 * @param {T} initValue 如果没有存储，则返回的初始值
*/
export function useLocalStorage <T extends {toString: () => string}> (
    key: LocalStorageKeys,
    initValue: T = null, 
    transform: (value: string) => T = (value) => value as unknown as T,
){

    const storageValue = ref<T>(initValue)
    /** 获取本地存储的值 */
    const getValue = () => {
        const storageData = localStorage.getItem(key)
        if (storageData !== null) {
            return transform(storageData)
        }
        else {
            return initValue
        }
    }
    /** 设置本地存储的值 */
    const setValue = (value: T) => {
        localStorage.setItem(key, value?.toString())
    }
    /** 清除本地存储的值 */
    const removeValue = () => {
        localStorage.removeItem(key)
    }
    /** 初始化 */
    storageValue.value = getValue()

    return {
        getValue,
        setValue,
        removeValue,
        storageValue,
    }
}