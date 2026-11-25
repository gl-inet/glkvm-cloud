/** 设备状态枚举 */
export enum DeviceStatusEnum {
    /** 在线 */
    Online = 'online',
    /** 离线 */
    Offline = 'offline',
}

/** 设备信息（列表） */
export interface DeviceInfo {
    client: '' | 'rtty-go'
    /** GL 硬件型号，其他主机为空 */
    model: string
    connectedTime: number
    ddns: string
    description: string
    deviceGroupId: number
    deviceGroupName: string
    id: number
    ip: string
    mac: string
    status: DeviceStatusEnum
    upTime: number
}

/** 设备列表查询条件 */
export interface DeviceQuery {
    searchText: string
    deviceGroupId: number
    onlyShowUnassigned: boolean
    status?: 'online' | 'offline'
    sortBy?: string
    order?: 'asc' | 'desc'
}

/** 接入码状态 */
export type RegistrationCodeStatus = 'unused' | 'used' | 'expired' | 'revoked'

/** 接入码（一次性注册码）记录 */
export interface RegistrationCode {
    id: number
    status: RegistrationCodeStatus
    createdAt: number
    expiresAt: number
    usedAt?: number
    usedMac?: string
    devid?: string
    revokedAt?: number
}

/** 新生成的接入码：明文只在这次返回里出现 */
export interface CreatedRegistrationCode {
    id: number
    accessCode: string
    expiresAt: number
}

/** 设备接入 mTLS 端口的地址 */
export interface DeviceAccess {
    /** 写进接入码的 host:port，空表示没配置接入地址 */
    addr: string
    port: string
    /** 部署文件早于 3.0：compose 没有对外开放这个端口 */
    outdatedBundle: boolean
    /** 平台仍接受旧 token 接入：固件低于 V1.11.0 的设备可以用脚本添加 */
    legacyScript: boolean
}

/** 固件低于 V1.11.0 的设备用的添加脚本 */
export interface LegacyScript {
    script: string
    /** 脚本里平台 CA 的 SHA-256 指纹 */
    caFingerprint: string
}
