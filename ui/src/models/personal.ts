/**
 * Personal Center 类型声明
 */

import { AuthProviderEnum, UserRoleEnum } from './userManage'

export interface PersonalProfile {
    id: number
    username: string
    displayName: string
    email: string
    role: UserRoleEnum
    authProvider: AuthProviderEnum
    /** 注册时间，unix 秒，0 表示未知 */
    registrationTime: number
    /** 最近登录时间，unix 秒，null 表示从未登录 */
    lastLoginTime: number | null
    totpEnabled: boolean
}

export interface Setup2faResp {
    secret: string
    otpauthUrl: string
}

export interface TrustedDevice {
    id: number
    deviceName: string
    ip: string
    /** unix 秒 */
    createdAt: number
    /** unix 秒 */
    lastUsedAt: number
    /** unix 秒 */
    expiresAt: number
}

export interface TrustedDeviceList {
    items: TrustedDevice[]
}
