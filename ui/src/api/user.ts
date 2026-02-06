/*
 * @Author: LPY
 * @Date: 2025-06-03 12:21:21
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 10:11:14
 * @FilePath: \glkvm-cloud\ui\src\api\user.ts
 * @Description: 用户相关请求api
 */

import request from './request'
import type { LoginParams, AuthConfig, UserInfo } from '@/models/user'

/** 登录 */
export function reqLogin (data: LoginParams) {
    return request<{ token: string }>({
        url: '/api/login',
        method: 'POST',
        data,
    })
}

/** 退出登录 */
export function reqLogout () {
    return request<void>({
        url: '/api/logout',
    })
}

/** 获取认证配置 */
export function reqAuthConfig () {
    return request<AuthConfig>({
        url: '/auth-config',
    })
}

/** 当前用户信息与权限 */
export function reqUserInfo () {
    return request<UserInfo>({
        url: '/api/me',
    })
}