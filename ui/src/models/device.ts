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
}

/** 执行命令参数 */
export interface ExecuteCommandParams {
    id: number
    group: string
    wait: number
    cmd: string
    params: string[]
    username: string
}

/** 执行命令表单参数 */
export interface ExecuteCommandFormData {
    username: string
    wait: number
    cmd: string
    params: string[]
}