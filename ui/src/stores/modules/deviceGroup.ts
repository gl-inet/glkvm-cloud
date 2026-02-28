/*
 * @Author: LPY
 * @Date: 2026-01-30 10:19:24
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-28 09:28:39
 * @FilePath: \glkvm-cloud\ui\src\stores\modules\deviceGroup.ts
 * @Description: 设备组有关的状态管理
 */
import { reqDeviceGroupList } from '@/api/deviceGroup'
import { DeviceGroup, DeviceGroupQuery } from '@/models/deviceGroup'
import { PageLink } from 'gl-web-main'
import { defineStore } from 'pinia'
import { computed, reactive, ref, watch } from 'vue'

/** 轮训获取设备列表的间隔时间 10s */
const GET_DEVICE_GROUP_POLLING_INTERVAL = 10 * 1000
/** 轮训获取设备列表的定时器 */
let getDeviceGroupListTimer: number
let pollingEnable = false

const DEVICE_VIEW_PAGE_SIZE = 20

export const useDeviceGroupStore = defineStore('deviceGroup', () => {
    const state = reactive({
        /** 设备组列表 */
        deviceGroupList: [] as DeviceGroup[],
        /** 完整设备列表 */
        completeDeviceGroupList: [] as DeviceGroup[],
        /** 获取设备列表的加载状态 */
        getDeviceGroupLoading: false,
        /** 设备列表的文字搜索 */
        searchText: '',
        /** 设备列表的用户组筛选条件 */
        userGroupId: undefined,
    })

    const pageLink = ref(new PageLink({ size: DEVICE_VIEW_PAGE_SIZE }))
  
    const handleSearch = (text: string) => {
        state.searchText = text
    }
    /** 计算设备列表的查询条件 */
    const computedDeviceGroupQuery = computed<DeviceGroupQuery>(() => {
        const query: DeviceGroupQuery = {
            searchText: state.searchText?.replaceAll(':','').toLowerCase(),
            userGroupId: state.userGroupId,
        }
        return query
    })
    /** 设备列表的分页展示数据 */
    const deviceGroupList= computed<DeviceGroup[]>(() => { 
        try {
            const { page, size } = pageLink.value
            return state.deviceGroupList.slice((page - 1) * size, page * size)
        } catch (error) {
            return []
        }    
    })
    /** 获取设备列表 */
    const getDeviceGroupList = async (isPolling = false) => {        
        try {
            !isPolling && (state.getDeviceGroupLoading = true)
            const res = await reqDeviceGroupList()
            pageLink.value.setTotal(res.data.items.length)
            state.deviceGroupList = res.data.items.filter(d => {
                return (d?.description?.indexOf(computedDeviceGroupQuery.value.searchText) > -1
                || d?.name?.indexOf(computedDeviceGroupQuery.value.searchText) > -1)
            }) || []
            state.completeDeviceGroupList = res.data.items || []
            !isPolling && (state.getDeviceGroupLoading = false)
        } catch (error) {
            state.deviceGroupList = []
            state.completeDeviceGroupList = []
            pageLink.value.setTotal(0)
            !isPolling && (state.getDeviceGroupLoading = false)
            console.error('Failed to fetch device list:', error)
        }
    }
    /** 设备列表是否有符合条件的展示数据 */
    const hasFilteredDevice = computed(() => {
        return deviceGroupList.value.length > 0
    })
    /** 停止轮询 */
    const stopPolling = () => {
        pollingEnable = false
        getDeviceGroupListTimer && clearTimeout(getDeviceGroupListTimer)
        getDeviceGroupListTimer = null
    }
    /** 轮询设备列表 */
    const startPolling = async () => { 
        stopPolling()
        pollingEnable = true
        getDeviceGroupListTimer = setTimeout(async () => {
            await getDeviceGroupList(true)
            pollingEnable && startPolling()
        }, GET_DEVICE_GROUP_POLLING_INTERVAL)
    }
    /** 监听设备列表的查询条件变化 */
    watch(computedDeviceGroupQuery, () => {
        getDeviceGroupList()
    })

    return {
        state,
        pageLink,
        deviceGroupList,
        hasFilteredDevice,
        getDeviceGroupList,
        handleSearch,
        startPolling,
        stopPolling,
    }
})