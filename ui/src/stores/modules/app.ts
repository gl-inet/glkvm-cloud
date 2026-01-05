/*
 * @Author: LPY
 * @Date: 2025-05-30 09:37:06
 * @LastEditors: LPY
 * @LastEditTime: 2026-01-05 14:35:02
 * @FilePath: \glkvm-cloud\ui\src\stores\modules\app.ts
 * @Description: app相关状态存储
 */
import { LocalStorageKeys, useLocalStorage } from '@/hooks/useLocalStorage'
import { baseTheme, createStyleInsert, darkTheme, replaceAntTheme, ThemeMode } from 'gl-web-main'
import type { ThemeConfig } from 'ant-design-vue/es/config-provider/context'
import { defineStore } from 'pinia'
import { computed, reactive } from 'vue'

export const useAppStore = defineStore('appGlobal', () => {
    // 从localStorage获取主题配置
    const {setValue: setThemeToStorage, getValue: getThemeFromStorage} = useLocalStorage(LocalStorageKeys.THEME_MODE_KEY, ThemeMode.LIGHT, (value) => {
        if (value === ThemeMode.DARK) {
            return ThemeMode.DARK
        }
        return ThemeMode.LIGHT
    })

    const state = reactive({
        /** 当前的主题模式 */
        themeMode: getThemeFromStorage(),
        /** 版本号 */
        version: (useLocalStorage(LocalStorageKeys.VERSION).getValue() as string)?.toUpperCase() || 'V1.0.0',
    })
    
    /** 获取主题模式 */
    const themeConfig = computed(() => {
        if (isDarkMode.value) {
            return darkTheme
        }
        return baseTheme
    })

    /** 判断是否是暗黑模式 */
    const isDarkMode = computed(() => {
        return state.themeMode === ThemeMode.DARK
    })

    /** 设置主题变量到页面 */
    const setThemeVarsToDocument = () => {
        createStyleInsert(themeConfig.value)
    }

    /** 初始化的时候就设置主题 */
    setThemeVarsToDocument()

    /** 获取antd 主题配置 */
    const antdTheme = computed<ThemeConfig['token']>(() => replaceAntTheme(themeConfig.value.content))

    /** 设置主题 */
    const setThemeMode = (theme: ThemeMode) => {
        state.themeMode = theme
        // 保存配置到localStorage
        setThemeToStorage(theme)
        // 设置主题变量到页面
        setThemeVarsToDocument()
    }

    // 侧边栏相关
    const sidebar = reactive({
        /** 侧边栏是否手动控制 */
        manualSetting: !!useLocalStorage(LocalStorageKeys.SIDEBAR_MANUAL_CONTROL_KEY).getValue(),
        /** 侧边栏是否自动展开 */
        automaticOpen: false,
        /** 侧边栏是否手动展开 */
        manualOpen: useLocalStorage(LocalStorageKeys.SIDEBAR_MANUAL_CONTROL_KEY).getValue() === 'open',
    })

    /** 侧边栏是否展开（最终控制） */
    const isCollapse = computed(() => {
        return sidebar.manualSetting ? !sidebar.manualOpen : !sidebar.automaticOpen
    })

    /** 手动切换侧边栏状态 */
    const manualToggleSidebar = () => {
        // 判断当前是否有已存储的展开/收缩状态
        if (useLocalStorage(LocalStorageKeys.SIDEBAR_MANUAL_CONTROL_KEY).getValue()) {
            if (sidebar.manualOpen) {
                useLocalStorage(LocalStorageKeys.SIDEBAR_MANUAL_CONTROL_KEY).setValue('close')
            } else {
                useLocalStorage(LocalStorageKeys.SIDEBAR_MANUAL_CONTROL_KEY).setValue('open')
            }
            sidebar.manualOpen = !sidebar.manualOpen
            sidebar.manualSetting = true
        } else {
            // 判断当前是自动展开还是收缩状态
            if (sidebar.automaticOpen) {
                useLocalStorage(LocalStorageKeys.SIDEBAR_MANUAL_CONTROL_KEY).setValue('close')
                sidebar.manualOpen = false
                sidebar.manualSetting = true
            } else {
                useLocalStorage(LocalStorageKeys.SIDEBAR_MANUAL_CONTROL_KEY).setValue('open')
                sidebar.manualOpen = true
                sidebar.manualSetting = true
            }
        }
    }

    /** 自动关闭侧边栏状态 */
    const autoCloseSidebar = () => {
        sidebar.automaticOpen = false
    }

    /** 自动开启侧边栏状态 */
    const autoOpenSidebar = () => {
        sidebar.automaticOpen = true
    }

    /** 重置侧边栏手动控制状态 */
    const resetManualSetting = () => {
        sidebar.manualSetting = false
    }

    /** 设置版本号 */
    const setVersion = (version: string) =>{
        state.version = version.toUpperCase()
        useLocalStorage(LocalStorageKeys.VERSION).setValue(version)
    }

    return { antdTheme, setThemeMode, state, sidebar, isCollapse, manualToggleSidebar, autoCloseSidebar, autoOpenSidebar, resetManualSetting, setVersion }
})