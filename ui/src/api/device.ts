/*
 * @Author: shufei.han
 * @Date: 2025-06-11 11:48:02
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 09:32:31
 * @FilePath: \glkvm-cloud\ui\src\api\device.ts
 * @Description: 设备相关API
 */
import { ExecuteCommandParams, type DeviceInfo } from '@/models/device'
import request, { httpService } from './request'

/** 获取设备列表 */
export const getDeviceListApi = (params?: { groupId: number }) => {
    return request<{ items: DeviceInfo[]}>({
        url: '/api/devices',
        params,
    })
}

/** 获取添加设备脚本 */
export const getAddDeviceScriptInfoApi = () => {
    return request({
        url: '/api/script-info',
    })
}

/** 执行命令 */
export const reqExecuteCommand = (data: ExecuteCommandParams) => {
    return httpService.post(`/cmd/${data.id}?group=${data.group}&wait=${data.wait}`, data)
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