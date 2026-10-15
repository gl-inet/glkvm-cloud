/*
 * @Author: shufei.han
 * @Date: 2025-06-11 11:48:02
 * @LastEditors: LPY
 * @LastEditTime: 2026-03-09 12:08:45
 * @FilePath: \glkvm-cloud\ui\src\api\device.ts
 * @Description: 设备相关API
 */
import { ExecuteCommandParams, type CreatedRegistrationCode, type DeviceInfo, type RegistrationCode } from '@/models/device'
import request from './request'
import axios from 'axios'

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

/** 生成接入码（每个 24 小时内有效、只能用一次） */
export const createRegistrationCodesApi = (count: number) => {
    return request<{ items: CreatedRegistrationCode[] }>({
        url: '/api/registration-codes',
        method: 'POST',
        data: { count },
    })
}

/** 最近的接入码 */
export const listRegistrationCodesApi = () => {
    return request<{ items: RegistrationCode[] }>({
        url: '/api/registration-codes',
    })
}

/** 作废未使用的接入码 */
export const revokeRegistrationCodeApi = (id: number) => {
    return request({
        url: `/api/registration-codes/${id}`,
        method: 'DELETE',
    })
}

/** 获取添加设备脚本（旧固件） */
export const getAddDeviceScriptInfoApi = () => {
    return request({
        url: '/api/script-info',
    })
}

/** 执行命令 */
// The /cmd endpoint is served directly by rttys and does NOT use the cloud
// `{code:"OK",...}` envelope: success is {code:0,stdout,stderr,devid}, an rtty error is
// {err:1xxx,msg}, and wait=0 returns an empty body. Routing it through httpService's
// interceptor would (a) reject even a successful command (code 0 !== "OK") and (b) pop a
// "Server Error Message" toast via showErrorMessage. Use a bare axios call to bypass the
// interceptor entirely; it is same-origin, so the `sid` auth cookie is still sent, and the
// caller reads the native body from res.data.
export const reqExecuteCommand = (data: ExecuteCommandParams) => {
    return axios.post(`/cmd/${data.id}?group=${data.group}&wait=${data.wait}`, data)
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