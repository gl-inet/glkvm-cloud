/*
 * @Author: LPY
 * @Date: 2026-02-03 12:07:45
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 10:56:54
 * @FilePath: \glkvm-cloud\ui\src\stores\modules\userGroupManage.ts
 * @Description: 用户组管理相关状态管理
 */

import { reqUserGroupList } from '@/api/userManage'
import { UserGroup, UserGroupManageQuery } from '@/models/userManage'
import { PageLink } from 'gl-web-main'
import { defineStore } from 'pinia'
import { computed, reactive, ref, watch } from 'vue'

/** 轮训获取用户列表的间隔时间 10s */
const GET_USER_GROUP_POLLING_INTERVAL = 10 * 1000
/** 轮训获取用户列表的定时器 */
let getUserGroupListTimer: number
let pollingEnable = false

const USER_GROUP_VIEW_PAGE_SIZE = 20

export const useUserGroupManageStore = defineStore('userGroupManage', () => {
    const state = reactive({
        /** 用户组列表 */
        userGroupList: [] as UserGroup[],
        /** 完整的用户组列表 */
        completeUserGroupList: [] as UserGroup[],
        /** 获取用户组列表的加载状态 */
        getUserGroupListLoading: false,
        /** 用户组列表的文字搜索 */
        searchText: '',
        /** 用户组列表的用户组筛选条件 */
        deviceGroupId: undefined,
    })

    const pageLink = ref(new PageLink({ size: USER_GROUP_VIEW_PAGE_SIZE }))
  
    const handleSearch = (text: string) => {
        state.searchText = text
    }
    /** 计算用户组列表的查询条件 */
    const computedUserGroupManageQuery = computed<UserGroupManageQuery>(() => {
        const query: UserGroupManageQuery = {
            searchText: state.searchText?.replaceAll(':','').toLowerCase(),
            deviceGroupId: state.deviceGroupId,
        }
        return query
    })
    /** 用户组列表的分页展示数据 */
    const userGroupList= computed<UserGroup[]>(() => { 
        try {
            const { page, size } = pageLink.value
            return state.userGroupList.slice((page - 1) * size, page * size)
        } catch (error) {
            return []
        }    
    })
    
    /** 获取用户组列表 */
    const getUserGroupList = async (isPolling = false) => {        
        try {
            !isPolling && (state.getUserGroupListLoading = true)
            const res = await reqUserGroupList()
            pageLink.value.setTotal(res.data.items.length)
            state.userGroupList = res.data.items.filter(d => {
                return (d?.id?.toString()?.indexOf(computedUserGroupManageQuery.value.searchText) > -1 
                || d?.userGroup?.indexOf(computedUserGroupManageQuery.value.searchText) > -1) && 
                (computedUserGroupManageQuery.value.deviceGroupId ? 
                    d.deviceGroupList.some(u => u.deviceGroupId === computedUserGroupManageQuery.value.deviceGroupId) : true)
            }) || []
            state.completeUserGroupList = res.data.items || []
            !isPolling && (state.getUserGroupListLoading = false)
        } catch (error) {
            state.userGroupList = []
            state.completeUserGroupList = []
            pageLink.value.setTotal(0)
            !isPolling && (state.getUserGroupListLoading = false)
            console.error('Failed to fetch device list:', error)
        }
    }
    /** 用户列表是否有符合条件的展示数据 */
    const hasFilteredUserGroup = computed(() => {
        return userGroupList.value.length > 0
    })
    /** 停止轮询 */
    const stopPolling = () => {
        pollingEnable = false
        getUserGroupListTimer && clearTimeout(getUserGroupListTimer)
        getUserGroupListTimer = null
    }
    /** 轮询设备列表 */
    const startPolling = async () => { 
        stopPolling()
        pollingEnable = true
        getUserGroupListTimer = setTimeout(async () => {
            await getUserGroupList(true)
            pollingEnable && startPolling()
        }, GET_USER_GROUP_POLLING_INTERVAL)
    }
    /** 监听设备列表的查询条件变化 */
    watch(computedUserGroupManageQuery, () => {
        getUserGroupList()
    })

    return {
        state,
        pageLink,
        userGroupList,
        hasFilteredUserGroup,
        getUserGroupList,
        handleSearch,
        startPolling,
        stopPolling,
    }
})