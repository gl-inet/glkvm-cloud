/*
 * @Author: LPY
 * @Date: 2026-02-04 10:14:15
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 10:17:54
 * @FilePath: \glkvm-cloud\ui\src\models\permission.ts
 * @Description: 权限相关类型声明
 */
export enum PermissionEnum {
  /** 显示设备列表 / 设备详情 */
  DEVICE_READ = 'device.read',
  /** 编辑 / 禁用 / 删除设备 */
  DEVICE_WRITE = 'device.write',
  /** 查看设备组 */
  DEVICE_GROUP_READ = 'device_group.read',
  /** 创建 / 编辑 / 删除设备组 */
  DEVICE_GROUP_WRITE = 'device_group.write',
  /** 查看用户组 */
  USER_GROUP_READ = 'user_group.read',
  /** 创建 / 编辑 / 删除用户组 */
  USER_GROUP_WRITE = 'user_group.write',
  /** 查看用户列表 */
  USER_READ = 'user.read',
  /** 创建 / 编辑 / 禁用用户 */
  USER_WRITE = 'user.write',
  /** 管理以下关系：• 用户 ↔ 用户组• 用户组 ↔ 设备组• 设备 ↔ 设备组 */
  RELATION_WRITE = 'relation.write',
  /** 查看设备事件日志 (admin only) */
  DEVICE_LOG_READ = 'device_log.read',
}