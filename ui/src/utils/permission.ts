/*
 * @Author: LPY
 * @Date: 2026-02-04 10:40:59
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 10:52:11
 * @FilePath: \glkvm-cloud\ui\src\utils\permission.ts
 * @Description: 权限相关方法
 */

import { PermissionEnum } from '@/models/permission'
import { useUserStore } from '@/stores/modules/user'

export const hasPermission = (permission: PermissionEnum | PermissionEnum[]) => {
    const permissions = useUserStore().userInfo?.permissions || []
    
    if (Array.isArray(permission)) {
        return permission.every(p => permissions.includes(p))
    }
    
    return permissions.includes(permission)
}