/*
 * @Author: LPY
 * @Date: 2025-05-30 09:44:40
 * @LastEditors: LPY
 * @LastEditTime: 2026-03-25 11:09:30
 * @FilePath: \glkvm-cloud\ui\src\projectInitialize\index.ts
 * @Description: 项目初始化的操作
 */
import type { App } from 'vue'
import plugin from '@/plugin'
import { initializeAllLanguage } from '@/lang'
import { installComponent } from './installComponent'
import loadAdvComponent from './loadAdvComponent'
import { installDirective } from './installDirective'
import { checkAndClearCache } from '@/utils/versionManager'

export default function (app: App ) {
    /** 加载插件 */
    plugin(app)

    /** 注册 AntD vue 组件 */
    loadAdvComponent(app)

    /** 注册自定义全局组件 */
    installComponent(app)

    /** 注册自定义全局指令 */
    installDirective(app)

    /** 初始化语言 */
    initializeAllLanguage()

    /** 检查并清理缓存 */
    checkAndClearCache()
}