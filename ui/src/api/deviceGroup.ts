/*
 * @Author: LPY
 * @Date: 2026-01-30 10:26:50
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-02 14:26:06
 * @FilePath: \glkvm-cloud\ui\src\api\deviceGroup.ts
 * @Description: 设备组相关API
 */

import { DeviceGroup } from '@/models/deviceGroup'
import request from './request'

/** 获取设备组列表 */
export const reqDeviceGroupList = () => {
    return request<{ items: DeviceGroup[] }>({
        url: '/api/device-groups',
    })
}

/** 获取用户组列表（用于“编辑设备组”时选择授权用户组） */
export const reqUserGroupListOptions = () => {
    return request<{ items: { userGroupId: number; name: string }[] }>({
        url: '/api/user-groups/options',
    })
}

/** 删除设备组 */
export const reqDeleteDeviceGroup = (groupId: number) => {
    return request({
        url: `/api/device-groups/${groupId}`,
        method: 'DELETE',
    })
}

/** 从设备组删除设备（批量） */
export const reqDeleteDevicesFromDeviceGroup = (groupId: number, data: { deviceIds: number[] }) => {
    return request({
        url: `/api/device-groups/${groupId}/devices`,
        method: 'DELETE',
        data,
    })
}

/** 编辑设备组 */
export const reqEditDeviceGroup = (groupId: number, data: { name: string, description?: string, userGroupIds?: number[] }) => {
    return request({
        url: `/api/device-groups/${groupId}`,
        method: 'PUT',
        data,
    })
}