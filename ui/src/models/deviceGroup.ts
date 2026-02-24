/*
 * @Author: LPY
 * @Date: 2026-01-30 10:22:49
 * @LastEditors: LPY
 * @LastEditTime: 2026-01-30 10:50:25
 * @FilePath: \glkvm-cloud\ui\src\models\deviceGroup.ts
 * @Description: 设备组相关类型声明
 */
export interface DeviceGroup {
    id: number
    name: string
    description: string
    userGroupList: {
      userGroupId: number
      userGroupName: string
    }[]
    deviceCount: number
}

export interface DeviceGroupQuery {
    searchText: string
    userGroupId: number
}