/*
 * @Author: LPY
 * @Date: 2025-06-19 09:58:12
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 10:51:18
 * @FilePath: \glkvm-cloud\ui\src\projectInitialize\installDirective.ts
 * @Description: 加载自定义指令
 */
import { App } from 'vue'
import ellipsis from '@/directive/ellipsis'

/** 全局注册自定义指令 */
const installDirective = function (app: App) {
    app.directive('ellipsis', ellipsis)
}

export {
    installDirective,
}