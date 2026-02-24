/*
 * @Author: LPY
 * @Date: 2026-02-02 15:19:27
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-03 14:48:24
 * @FilePath: \glkvm-cloud\ui\src\api\userManage.ts
 * @Description: 用户及用户组相关API
 */

import request from './request'
import { UserGroupList, UserList, UserRoleEnum } from '@/models/userManage'

/** 获取用户列表 */
export function reqUserList () {
    return request<UserList>({
        url: '/api/users',
    })
}

/** 新建用户 */
export function reqCreateUser (data: 
  { role: UserRoleEnum, username: string, description: string, password: string, repassword: string, userGroupIds: number[] },
) {
    return request({
        url: '/api/users',
        method: 'POST',
        data,
    })
}

/** 删除用户 */
export function reqDeleteUser (userId: number) {
    return request({
        url: `/api/users/${userId}`,
        method: 'DELETE',
    })
}

/** 编辑用户 */
export function reqEditUser (userId: number, data: 
  { role: UserRoleEnum, username: string, description: string, password: string, repassword: string, userGroupIds: number[] },
) {
    return request({
        url: `/api/users/${userId}`,
        method: 'PUT',
        data,
    })
}

/** 新建用户组 */
export function reqCreateUserGroup (data: { name: string, description: string }) {
    return request<{ id: number }>({
        url: '/api/user-groups',
        method: 'POST',
        data,
    })
}

/** 获取用户组列表 */
export function reqUserGroupList () {
    return request<UserGroupList>({
        url: '/api/user-groups',
    })
}

/** 编辑用户组 */
export function reqEditUserGroup (userGroupId: number, data: { name: string, description: string }) {
    return request({
        url: `/api/user-groups/${userGroupId}`,
        method: 'PUT',
        data,
    })
}

/** 关联设备组 */
export function reqAssociateDeviceGroup (userGroupId: number, data: { deviceGroupIds: number[] }) {
    return request({
        url: `/api/user-groups/${userGroupId}/device-groups`,
        method: 'PUT',
        data,
    })
}

/** 删除用户组 */
export function reqDeleteUserGroup (userGroupId: number) {
    return request({
        url: `/api/user-groups/${userGroupId}`,
        method: 'DELETE',
    })
}