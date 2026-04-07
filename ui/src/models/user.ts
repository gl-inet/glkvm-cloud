/*
 * @Author: LPY
 * @Date: 2025-06-09 16:37:00
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 11:28:24
 * @FilePath: \glkvm-cloud\ui\src\models\user.ts
 * @Description: 用户相关类型声明
 */

import { PermissionEnum } from './permission'
import { UserRoleEnum } from './userManage'

/** 登录参数 (Login parameters) */
export interface LoginParams {
    username: string;
    password: string;
    authMethod?: 'ldap' | 'legacy';
    /** 6 位 TOTP 验证码，2FA 启用且未绑定信任设备时需提供 */
    totpCode?: string;
    /** 是否将本浏览器加入信任设备（30 天免 2FA） */
    rememberDevice?: boolean;
}

/** 登录响应 (Login response) */
export interface LoginResp {
    token?: string;
    /** 后端要求 2FA 时为 true，前端应弹出 TOTP 输入框 */
    twoFactorRequired?: boolean;
}

/** 认证配置 (Authentication configuration) */
export interface AuthConfig {
    ldapEnabled: boolean;
    legacyPassword: boolean;
    oidcEnabled: boolean;
    kvmCloudVersion: string;
}

/** 当前用户信息与权限 (Current user information and permissions) */
export interface UserInfo {
    permissions: PermissionEnum[];
    user: {
        displayName: string;
        id: number;
        role: UserRoleEnum;
        username: string;
    }
}