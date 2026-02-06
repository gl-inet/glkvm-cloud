/*
 * @Author: LPY
 * @Date: 2026-02-02 15:13:17
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-06 14:07:48
 * @FilePath: \glkvm-cloud\ui\src\models\userManage.ts
 * @Description: 用户管理相关类型声明
 */

import { t } from '@/hooks/useLanguage'

export interface UserManageQuery {
    searchText: string
    userGroupId: number
}

export enum UserRoleEnum {
    ADMIN = 'admin',
    USER = 'user',
}

export const UserRoleLabelMap = new Map([
    [UserRoleEnum.ADMIN, t('user.admin')],
    [UserRoleEnum.USER, t('user.user')],
])

export interface UserManage {
    id: number
    username: string
    role: UserRoleEnum
    description: string
    isSystem: boolean
    userGroupList: {
      userGroupId: number
      userGroupName: string
    }[]
}

export interface UserList {
    items: UserManage[]
}

export interface UserGroup {
    id: number
    userGroup: string
    description: string
    userCount: number
    deviceGroupList: {
        deviceGroupId: number
        deviceGroupName: string
    }[]
}

export interface UserGroupList {
    items: UserGroup[]
}

export interface UserGroupManageQuery {
    searchText: string
    deviceGroupId: number
}