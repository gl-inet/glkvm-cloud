<!--
 * @Author: LPY
 * @Date: 2025-05-29 17:09:32
 * @LastEditors: shufei.han
 * @LastEditTime: 2025-06-10 12:27:00
 * @FilePath: \kvm-cloud-frontend\src\App.vue
 * @Description: APP 入口文件
-->
<template>
    <GlConfigProvider 
        :t="$t"
        :locales="null"
        :locale="currentLang"
        :antConfigProviderProps="{ theme: themeConfig }"
        :themeMode="appStore.state.themeMode">
        <AConfigProvider :locale="antLocale" :theme="themeConfig">
            <RouterView />
        </AConfigProvider>
    </GlConfigProvider>
</template>

<script setup lang="ts">
import useLanguage from './hooks/useLanguage'
import enUS from 'ant-design-vue/es/locale/en_US'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import { computed } from 'vue'
import { useAppStore } from './stores/modules/app'
import { RouterView } from 'vue-router'
import type { ThemeConfig } from 'ant-design-vue/es/config-provider/context'
import { ConfigProvider as GlConfigProvider } from 'gl-web-main/components'
import { Languages } from 'gl-web-main'

const appStore = useAppStore()

// 当前语言
const { currentLang } = useLanguage()

// ant d的语言包和我们系统语言的映射关系
const localeMap = new Map([
    [Languages.ZH, zhCN],
    [Languages.EN, enUS],
])

// ant d的语言包
const antLocale = computed(() => localeMap.get(currentLang.value))

// 主题配置
const themeConfig = computed<ThemeConfig>(() => {
    return { token: appStore.antdTheme }
})
</script>

<style lang="scss">
* {
    box-sizing: border-box;
}
</style>