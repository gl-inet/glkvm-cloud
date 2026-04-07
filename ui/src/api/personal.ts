/**
 * Personal Center 相关 API
 */

import request from './request'
import type { PersonalProfile, Setup2faResp, TrustedDeviceList } from '@/models/personal'

/** 获取个人信息 */
export function reqGetProfile () {
    return request<PersonalProfile>({
        url: '/api/me/profile',
    })
}

/** 更新显示名（description） */
export function reqUpdateProfile (data: { displayName: string }) {
    return request({
        url: '/api/me/profile',
        method: 'PUT',
        data,
    })
}

/** 启动 2FA 注册：生成 secret 与 otpauth URL（不持久化） */
export function reqSetup2fa () {
    return request<Setup2faResp>({
        url: '/api/me/2fa/setup',
        method: 'POST',
    })
}

/** 启用 2FA：提交 secret 与 6 位验证码完成绑定 */
export function reqEnable2fa (data: { secret: string, code: string }) {
    return request({
        url: '/api/me/2fa/enable',
        method: 'POST',
        data,
    })
}

/** 关闭 2FA：需要提供当前 6 位验证码 */
export function reqDisable2fa (data: { code: string }) {
    return request({
        url: '/api/me/2fa/disable',
        method: 'POST',
        data,
    })
}

/** 信任设备列表 */
export function reqListTrustedDevices () {
    return request<TrustedDeviceList>({
        url: '/api/me/2fa/trusted-devices',
    })
}

/** 撤销单个信任设备 */
export function reqRevokeTrustedDevice (id: number) {
    return request({
        url: `/api/me/2fa/trusted-devices/${id}`,
        method: 'DELETE',
    })
}
