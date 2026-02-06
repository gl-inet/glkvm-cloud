/*
 * @Author: LPY
 * @Date: 2026-02-02 15:00:13
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-03 12:08:51
 * @FilePath: \glkvm-cloud\ui\src\stores\modules\userManage.ts
 * @Description: 用户管理相关状态管理
 */
import { reqUserList } from '@/api/userManage'
import { UserManage, UserManageQuery, UserRoleEnum } from '@/models/userManage'
import { PageLink } from 'gl-web-main'
import { defineStore } from 'pinia'
import { computed, reactive, ref, watch } from 'vue'

/** 轮训获取用户列表的间隔时间 10s */
const GET_USER_POLLING_INTERVAL = 10 * 1000
/** 轮训获取用户列表的定时器 */
let getUserListTimer: number
let pollingEnable = false

const USER_VIEW_PAGE_SIZE = 20

export const useUserManageStore = defineStore('userManage', () => {
    const state = reactive({
        /** 用户列表 */
        userList: [] as UserManage[],
        /** 完整用户列表 */
        completeUserList: [] as UserManage[],
        /** 获取用户列表的加载状态 */
        getUserListLoading: false,
        /** 用户列表的文字搜索 */
        searchText: '',
        /** 用户列表的用户组筛选条件 */
        userGroupId: undefined,
    })

    const pageLink = ref(new PageLink({ size: USER_VIEW_PAGE_SIZE }))
  
    const handleSearch = (text: string) => {
        state.searchText = text
    }
    /** 计算用户列表的查询条件 */
    const computedUserManageQuery = computed<UserManageQuery>(() => {
        const query: UserManageQuery = {
            searchText: state.searchText?.replaceAll(':','').toLowerCase(),
            userGroupId: state.userGroupId,
        }
        return query
    })
    /** 用户列表的分页展示数据 */
    const userList= computed<UserManage[]>(() => { 
        try {
            const { page, size } = pageLink.value
            return state.userList.slice((page - 1) * size, page * size)
        } catch (error) {
            return []
        }    
    })

    /** 是否为只有一个ADMIN */
    const isOnlyOneAdmin = computed(() => {
        return state.completeUserList.filter(u => u.role === UserRoleEnum.ADMIN).length === 1
    })
    
    /** 获取用户列表 */
    const getUserList = async (isPolling = false) => {        
        try {
            !isPolling && (state.getUserListLoading = true)
            const res = await reqUserList()
            pageLink.value.setTotal(res.data.items.length)
            state.userList = res.data.items.filter(d => {
                return (d?.id?.toString().toLowerCase()?.indexOf(computedUserManageQuery.value.searchText) > -1 
                || d?.username?.toLowerCase()?.indexOf(computedUserManageQuery.value.searchText) > -1) && 
                (computedUserManageQuery.value.userGroupId ? d.userGroupList.some(u => u.userGroupId === computedUserManageQuery.value.userGroupId) : true)
            }) || []
            state.completeUserList = res.data.items || []
            !isPolling && (state.getUserListLoading = false)
        } catch (error) {
            state.userList = []
            state.completeUserList = []
            pageLink.value.setTotal(0)
            !isPolling && (state.getUserListLoading = false)
            console.error('Failed to fetch device list:', error)
        }
    }
    /** 用户列表是否有符合条件的展示数据 */
    const hasFilteredUser = computed(() => {
        return userList.value.length > 0
    })
    /** 停止轮询 */
    const stopPolling = () => {
        pollingEnable = false
        getUserListTimer && clearTimeout(getUserListTimer)
        getUserListTimer = null
    }
    /** 轮询设备列表 */
    const startPolling = async () => { 
        stopPolling()
        pollingEnable = true
        getUserListTimer = setTimeout(async () => {
            await getUserList(true)
            pollingEnable && startPolling()
        }, GET_USER_POLLING_INTERVAL)
    }
    /** 监听设备列表的查询条件变化 */
    watch(computedUserManageQuery, () => {
        getUserList()
    })

    return {
        state,
        pageLink,
        userList,
        hasFilteredUser,
        isOnlyOneAdmin,
        getUserList,
        handleSearch,
        startPolling,
        stopPolling,
    }
})