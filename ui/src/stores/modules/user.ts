/*
 * @Author: LPY
 * @Date: 2025-05-29 18:43:46
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 11:54:46
 * @FilePath: \glkvm-cloud\ui\src\stores\modules\user.ts
 * @Description: 用户相关状态存储
 */

import { ref } from 'vue'
import { defineStore } from 'pinia'
import { LocalStorageKeys, useLocalStorage } from '@/hooks/useLocalStorage'
import { reqLogin, reqLogout, reqUserInfo} from '@/api/user'
import router from '@/router'
import { UserInfo, type LoginParams } from '@/models/user'
import { useAppStore } from './app'
import { getCookieToken, removeCookieToken, setCookieToken } from '@/utils/auth'

export const useUserStore = defineStore('user', () => {
    // 用户信息
    const userInfo = ref<UserInfo>(null)
    // token
    const token = ref(getCookieToken())

    /** 设置token */
    const setToken = (newToken: string) => {
        token.value = newToken
        setCookieToken(newToken)
    }

    /** 清除token */
    const clearToken = () => {
        token.value = ''
        removeCookieToken()
    }

    /** 获取用户信息 */
    const fetchUserInfo = async () => {
        if (!token.value) {
            userInfo.value = null
            return
        }
        try {
            const res = await reqUserInfo()
            userInfo.value = res.data
        } catch (error) {
            clearToken()
            throw error
        }
    }

    /** 登录 */
    const login = async (credentials: LoginParams) => {
        // 准备登录参数 (Prepare login parameters)
        const params: LoginParams = {
            username: credentials.username,
            password: credentials.password,
            authMethod: credentials.authMethod,
        }
        
        const data = await reqLogin(params)
        setToken(data.data.token)
        fetchUserInfo()
    }

    /** 合并登出方法，不可导出使用 */
    const logout = () => {
        // 清除token
        clearToken()
        
        // 清除侧边栏状态
        useAppStore().resetManualSetting()
        useLocalStorage(LocalStorageKeys.SIDEBAR_MANUAL_CONTROL_KEY).removeValue()

        // 登出
        router.push({ path: '/login' })
    }
  
    /** 自动登出 */
    const autoLogout = () => {
        logout()
    }

    /** 手动登出 */
    const manualLogout = async () => {
        try {
            await reqLogout()
            logout()
        } catch (error) {
            console.log(error)
            // 即使退出请求失败也继续本地退出 (Continue local logout even if logout request fails)
            logout()
        }
    }

    return {
        userInfo,
        token,
        setToken,
        clearToken,
        fetchUserInfo,
        login,
        autoLogout,
        manualLogout,
    }
})