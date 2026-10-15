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

/** 执行命令参数 */
export interface ExecuteCommandParams {
    id: string
    group: string
    wait: number
    cmd: string
    params: string[]
    username: string
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

/** 执行命令表单参数 */
export interface ExecuteCommandFormData {
    username: string
    wait: number
    cmd: string
    params: string[]
}