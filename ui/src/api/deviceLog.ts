/**
 * Device Event Logs API
 */

import request from './request'

export interface DeviceEventLog {
    id: number
    deviceMac: string
    eventType: string
    actorName: string
    clientIp: string
    detail: string
    createdAt: number
    endedAt: number
}

export interface ListDeviceEventLogsResp {
    items: DeviceEventLog[]
    total: number
    page: number
    pageSize: number
}

export interface DeviceEventLogQuery {
    mac?: string
    types?: string[]
    from?: number
    to?: number
    page: number
    pageSize: number
}

/** 设备事件日志列表查询 */
export function reqListDeviceEventLogs (q: DeviceEventLogQuery) {
    return request<ListDeviceEventLogsResp>({
        url: '/api/device-event-logs',
        params: {
            mac: q.mac || undefined,
            types: q.types && q.types.length > 0 ? q.types.join(',') : undefined,
            from: q.from || undefined,
            to: q.to || undefined,
            page: q.page,
            pageSize: q.pageSize,
        },
    })
}
