/*
 * @Author: shufei.han
 * @Date: 2025-06-11 11:48:02
 * @LastEditors: LPY
 * @LastEditTime: 2026-03-09 12:08:45
 * @FilePath: \glkvm-cloud\ui\src\api\device.ts
 * @Description: 设备相关API
 */
import {
    type CreatedRegistrationCode, type DeviceAccess, type DeviceInfo, type LegacyScript, type RegistrationCode,
} from '@/models/device'
import request from './request'

/** 获取设备列表（服务端分页：page/pageSize；q 为搜索词；unassigned 仅未分配） */
export const getDeviceListApi = (params?: {
    groupId?: number
    sortBy?: string
    order?: 'asc' | 'desc'
    page?: number
    pageSize?: number
    q?: string
    unassigned?: boolean
    status?: 'online' | 'offline'
}) => {
    return request<{ items: DeviceInfo[], page: number, pageSize: number, total: number }>({
        url: '/api/devices',
        params,
    })
}

/** 设备接入地址和端口，以及部署文件是否过旧 */
export const getDeviceAccessApi = () => {
    return request<DeviceAccess>({ url: '/api/device-access' })
}

/** 固件低于 V1.11.0 的设备用的添加脚本：里面有平台 token，只有管理员能取 */
export const getLegacyScriptApi = () => {
    return request<LegacyScript>({ url: '/api/script-info' })
}

/** 生成接入码（每个 24 小时内有效、只能用一次） */
export const createRegistrationCodesApi = (count: number) => {
    return request<{ items: CreatedRegistrationCode[] }>({
        url: '/api/registration-codes',
        method: 'POST',
        data: { count },
    })
}

/** 最近的接入码；传 ids 时只查这几个（添加设备弹窗等待设备接入时用） */
export const listRegistrationCodesApi = (ids?: number[]) => {
    return request<{ items: RegistrationCode[] }>({
        url: '/api/registration-codes',
        params: ids?.length ? { ids: ids.join(',') } : undefined,
    })
}

/** 作废未使用的接入码 */
export const revokeRegistrationCodeApi = (id: number) => {
    return request({
        url: `/api/registration-codes/${id}`,
        method: 'DELETE',
    })
}

/** 修改描述 */
export const reqEditDescription = (deviceId: number, data: { description: string }) => {
    return request({
        url: `/api/devices/${deviceId}`,
        method: 'PUT',
        data,
    })
}

/** 删除设备 */
export const reqDeleteDevice = (deviceId: number) => {
    return request({
        url: `/api/devices/${deviceId}`,
        method: 'DELETE',
    })
}

/** 获取设备组列表（用于“设备移动至设备组”的下拉） */
export const reqDeviceGroupListOptions = () => {
    return request<{ items: { groupId: number, name: string }[] }>({
        url: '/api/device-groups/options',
    })
}

/** 将设备移动至设备组（批量） */
export const reqMoveDevicesToDeviceGroup = (data: { deviceIds: number[], groupId: number }) => {
    return request({
        url: '/api/devices/move-to-device-group',
        method: 'POST',
        data,
    })
}

/** 添加设备组 */
export const reqAddDeviceGroup = (data: { name: string, description?: string, userGroupIds?: number[], deviceIds?: number[] }) => {
    return request<{ id: number }>({
        url: '/api/device-groups',
        method: 'POST',
        data,
    })
}