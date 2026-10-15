/*
 * @Author: LPY
 * @Date: 2026-02-02 15:13:17
 * @LastEditors: LPY
 * @LastEditTime: 2026-03-25 10:09:43
 * @FilePath: \glkvm-cloud\ui\src\models\userManage.ts
 * @Description: 用户管理相关类型声明
 */

export interface UserManageQuery {
    searchText: string
    userGroupId: number
}

export enum UserRoleEnum {
    ADMIN = 'admin',
    USER = 'user',
}

export const UserRoleLabelMap = new Map([
    [UserRoleEnum.ADMIN, 'user.admin'],
    // The role is shown as "operator": regular accounts here operate the
    // devices of their groups. 'user.user' stays the generic word for accounts.
    [UserRoleEnum.USER, 'user.roleOperator'],
])

export enum AuthProviderEnum {
    LOCAL = 'local',
    LDAP = 'ldap',
    OIDC = 'oidc',
}

export const AuthProviderLabelMap = new Map([
    [AuthProviderEnum.LOCAL, 'login.local'],
    [AuthProviderEnum.LDAP, 'login.ldap'],
    [AuthProviderEnum.OIDC, 'login.oidc'],
])

export interface UserManage {
    id: number
    username: string
    role: UserRoleEnum
    description: string
    isSystem: boolean
    authProvider: AuthProviderEnum,
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