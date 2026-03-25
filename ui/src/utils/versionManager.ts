/*
 * @Author: LPY
 * @Date: 2026-03-25 11:01:34
 * @LastEditors: LPY
 * @LastEditTime: 2026-03-25 11:15:12
 * @FilePath: \glkvm-cloud\ui\src\utils\versionManager.ts
 * @Description: 版本管理工具，主要用于清理缓存
 */

import { LocalStorageKeys, useLocalStorage } from '@/hooks/useLocalStorage'

const APP_VERSION = '2.4.0' // 当前应用版本
const CACHE_KEYS_TO_CLEAR = [LocalStorageKeys.DEVICE_LIST_COLUMNS_KEY] // 需要清理的缓存key

export function checkAndClearCache () {
    const cachedVersion = useLocalStorage(LocalStorageKeys.APP_VERSION_KEY).getValue()
  
    if (cachedVersion !== APP_VERSION) {
        // 版本不一致，清理指定缓存
        CACHE_KEYS_TO_CLEAR.forEach(key => {
            useLocalStorage(key).removeValue()
        })
    
        // 更新版本号
        useLocalStorage(LocalStorageKeys.APP_VERSION_KEY).setValue(APP_VERSION)
    
        console.log(`缓存已清理，版本从 ${cachedVersion} 升级到 ${APP_VERSION}`)
        return true
    }
  
    return false
}