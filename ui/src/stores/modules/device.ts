/*
 * @Author: shufei.han
 * @Date: 2025-06-10 16:46:00
 * @LastEditors: LPY
 * @LastEditTime: 2026-03-09 12:17:03
 * @FilePath: \glkvm-cloud\ui\src\stores\modules\device.ts
 * @Description: 设备有关的状态管理
 */
import { getDeviceListApi } from '@/api/device'
import { type DeviceInfo, type DeviceQuery } from '@/models/device'
import { PageLink } from 'gl-web-main'
import { defineStore } from 'pinia'
import { computed, reactive, ref, watch } from 'vue'

/** 轮训获取设备列表的间隔时间 10s */
const GET_DEVICE_POLLING_INTERVAL = 10 * 1000
/** 轮训获取设备列表的定时器 */
let getDeviceListTimer: number
let pollingEnable = false

const DEVICE_VIEW_PAGE_SIZE = 50

export const useDeviceStore = defineStore('device', () => {
    const state = reactive({
        /** 设备列表 */
        deviceList: [] as DeviceInfo[],
        /** 完整设备列表 */
        completeDeviceList: [] as DeviceInfo[],
        /** 获取设备列表的加载状态 */
        getDeviceLoading: false,
        /** 设备列表的文字搜索 */
        searchText: '',
        /** 设备列表的设备组筛选条件 */
        deviceGroupId: undefined,
        /** 是否仅显示未分配项 */
        onlyShowUnassigned: false,
        /** 在线状态筛选：undefined=全部 / 'online' / 'offline' */
        status: undefined as 'online' | 'offline' | undefined,
        /** 这个字段存储是否有设备，因为UI上没有设备和没有筛选出来的设备是对应不同的展示画面的 */
        hasDevice: false,
        /** 添加设备弹窗是否打开：弹窗挂在设备页上，首台设备上线、无设备页切到列表时不会被关掉 */
        addDeviceOpen: false,
        /** 排序字段 */
        sortBy: undefined,
        /** 排序方式 */
        order: undefined,
    })

    const pageLink = ref(new PageLink({ size: DEVICE_VIEW_PAGE_SIZE }))
  
    const handleSearch = (text: string) => {
        state.searchText = text
    }
    /** 计算设备列表的查询条件 */
    const computedDeviceQuery = computed<DeviceQuery>(() => {
        const query: DeviceQuery = {
            searchText: state.searchText?.replaceAll(':','').toLowerCase(),
            deviceGroupId: state.deviceGroupId,
            onlyShowUnassigned: state.onlyShowUnassigned,
            status: state.status,
            sortBy: state.sortBy,
            order: state.order,
        }
        return query
    })
    /** 设备列表展示数据（服务端已分页，直接展示当前页） */
    const deviceList = computed<DeviceInfo[]>(() => state.deviceList)
    /** 获取设备列表（服务端分页 + 搜索/筛选下推后端） */
    const getDeviceList = async (isPolling = false, isGetAll = false) => {
        try {
            !isPolling && (state.getDeviceLoading = true)
            const res = await getDeviceListApi({
                page: pageLink.value.page,
                pageSize: pageLink.value.size,
                q: computedDeviceQuery.value.searchText || undefined,
                groupId: computedDeviceQuery.value.deviceGroupId,
                unassigned: computedDeviceQuery.value.onlyShowUnassigned || undefined,
                status: computedDeviceQuery.value.status || undefined,
                sortBy: computedDeviceQuery.value.sortBy,
                order: computedDeviceQuery.value.order,
            })
            const total = res.data.total ?? res.data.items.length
            pageLink.value.setTotal(total)
            // hasDevice 区分“账号一台设备都没有(引导页)”和“筛选无结果(空表格)”：
            // 只有无筛选的首次加载(isGetAll)可置 false，之后只升不降。
            if (isGetAll) {
                state.hasDevice = total > 0
            } else if (total > 0) {
                state.hasDevice = true
            }
            state.deviceList = res.data.items || []
            state.completeDeviceList = res.data.items || []
            !isPolling && (state.getDeviceLoading = false)
        } catch (error) {
            state.deviceList = []
            state.completeDeviceList = []
            pageLink.value.setTotal(0)
            !isPolling && (state.getDeviceLoading = false)
            console.error('Failed to fetch device list:', error)
        }
    }
    /** 设备列表是否有符合条件的展示数据 */
    const hasFilteredDevice = computed(() => {
        return deviceList.value.length > 0
    })
    /** 停止轮询 */
    const stopPolling = () => {
        pollingEnable = false
        getDeviceListTimer && clearTimeout(getDeviceListTimer)
        getDeviceListTimer = null
    }
    /** 轮询设备列表 */
    const startPolling = async () => { 
        stopPolling()
        pollingEnable = true
        getDeviceListTimer = setTimeout(async () => {
            await getDeviceList(true)
            pollingEnable && startPolling()
        }, GET_DEVICE_POLLING_INTERVAL)
    }
    /** 翻页（服务端分页）：页码变化即拉取当前页 */
    watch(() => pageLink.value.page, () => {
        getDeviceList()
    })
    /** 查询条件变化（搜索/组/未分配/排序）：重置到第 1 页。
     * 若已在第 1 页则直接拉取，否则改页码由上面的页码 watch 触发，避免重复请求。 */
    watch(computedDeviceQuery, () => {
        if (pageLink.value.page !== 1) {
            pageLink.value.changePage(1)
        } else {
            getDeviceList()
        }
    })

    return {
        state,
        pageLink,
        deviceList,
        hasFilteredDevice,
        getDeviceList,
        handleSearch,
        startPolling,
        stopPolling,
    }
})