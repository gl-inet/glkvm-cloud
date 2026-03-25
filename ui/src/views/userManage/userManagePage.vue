<!--
 * @Author: LPY
 * @Date: 2026-02-02 14:32:56
 * @LastEditors: LPY
 * @LastEditTime: 2026-03-25 10:10:53
 * @FilePath: \glkvm-cloud\ui\src\views\userManage\userManagePage.vue
 * @Description: 用户管理页
-->
<template>
    <div class="user-manage-list">
        <div class="user-manage-list-content">
            <BaseLoadingContainer :spinning="userManageStore.state.getUserListLoading">
                <div class="user-manage-list-view full-height">
                    <div class="header">
                        <div>
                            <a-input-search
                                :placeholder="$t('device.searchTip')"
                                style="width: 316px; margin-right: 8px;"
                                @search="userManageStore.handleSearch"
                            />
                
                            <ASelect 
                                v-model:value="userManageStore.state.userGroupId"
                                allowClear
                                :placeholder="$t('device.allAssociatedUserGroups')"
                                style="width: 224px;"
                            >
                                <ASelectOption v-for="item in state.userGroupList" :key="item.userGroupId" :value="item.userGroupId">
                                    {{ item.name }}
                                </ASelectOption>
                            </ASelect>
                        </div>
            
                        <div>
                            <BaseButton type="primary" size="middle" @click="state.addUserOpen = true">
                                <GlSvg name="gl-icon-plus-regular" style="color: var(--gl-color-text-white);margin-right: 4px;"></GlSvg>
                                {{ $t('user.addUser') }}
                            </BaseButton>
                        </div>
                    </div>
                    <div class="content">
                        <BaseTable
                            :data-source="userManageStore.userList"
                            :columns="userColumns"
                        >
                            <template #role="{ record }">
                                <BaseTag v-if="record.role === UserRoleEnum.USER" primary>{{ $t(UserRoleLabelMap.get(record.role)) }}</BaseTag>
                                <BaseTag 
                                    v-else-if="record.role === UserRoleEnum.ADMIN"
                                    style="background-color: var(--gl-color-warning-primary);color: var(--gl-color-warning-background);"
                                >{{ $t(UserRoleLabelMap.get(record.role)) }}</BaseTag>
                            </template>
                            <template #authProvider="{ record }">
                                {{ $t(AuthProviderLabelMap.get(record.authProvider || AuthProviderEnum.LOCAL)) }}
                            </template>
                            <template #userGroupList="{ record }">
                                <div class="groups-a">
                                    <a 
                                        v-for="item in record.userGroupList.slice(0, 2)"
                                        :key="item.userGroupId"
                                        rel="noopener noreferrer"
                                        @click="jumpToUserPage(item.userGroupId)"
                                    >{{ item.userGroupName }}</a>

                                    <BaseDropdownSelect
                                        v-if="record.userGroupList?.length > 2"
                                        :options="computedUserGroupDropDownOptions(record.userGroupList)"
                                        @update:value="(v) => jumpToUserPage(v)"
                                    >
                                        <a 
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            class="groups-a-dropdown">
                                            ...+{{ computedUserGroupDropDownOptions(record.userGroupList).length }}
                                        </a>
                                    </BaseDropdownSelect>
                                </div>
                            </template>
                            <template #description="{ record }">
                                <div v-ellipsis>
                                    {{ record.description }}
                                </div>
                            </template>
                            <template #action="{ record }">
                                <div class="flex-start">
                                    <a 
                                        rel="noopener noreferrer"
                                        style="margin-right: 16px;"
                                        @click="handleAction(DeviceGroupActions.EDIT, record)">{{ $t('common.edit') }}</a>
                                    <Tooltip>
                                        <template v-if="userManageStore.isOnlyOneAdmin && record.role === UserRoleEnum.ADMIN" #title>
                                            {{ $t('user.deleteOnlyOneAdminTips') }}
                                        </template>
                                        <a 
                                            rel="noopener noreferrer"
                                            style="color: var(--gl-color-error-primary);"
                                            :class="[{'disabled': (userManageStore.isOnlyOneAdmin && record.role === UserRoleEnum.ADMIN) || 
                                                record.isSystem || record.id === userStore.userInfo?.user?.id
                                            }]"
                                            @click="handleAction(DeviceGroupActions.DELETE, record)">{{ $t('common.delete') }}</a>
                                    </Tooltip>
                                </div>
                            </template>
                        </BaseTable>
                    </div>
                    <div class="pagination flex-end items-end">
                        <BasePagination
                            :total="userManageStore.pageLink.total"
                            :pageSize="userManageStore.pageLink.size"
                            v-model:current="page" />
                    </div>
                </div>
            </BaseLoadingContainer>
        </div>

        <!-- 添加用户弹窗 -->
        <AddUserDialog 
            v-model:open="state.addUserOpen"
            @handleApply="addUserApply"
        />

        <!-- 编辑用户弹窗 -->
        <EditUserDialog
            v-model:open="state.editUserOpen"
            :currentUser="state.currentUser"
            @handleApply="editUserApply"
        />
    </div>
</template>

<script setup lang="ts">
import { reqUserGroupListOptions } from '@/api/deviceGroup'
import BaseLoadingContainer from '@/components/base/baseLoadingContainer.vue'
import BasePagination from '@/components/base/basePagination.vue'
import BaseTable from '@/components/base/baseTable.vue'
import { t } from '@/hooks/useLanguage'
import { AuthProviderEnum, AuthProviderLabelMap, UserManage, UserRoleEnum, UserRoleLabelMap } from '@/models/userManage'
import { useUserManageStore } from '@/stores/modules/userManage'
import { message, TableColumnType, Tooltip } from 'ant-design-vue'
import { baseCustomModal, SelectOptions } from 'gl-web-main'
import { BaseDropdownSelect, BaseTag, GlSvg } from 'gl-web-main/components'
import { computed, onBeforeUnmount, onMounted, reactive } from 'vue'
import AddUserDialog from './components/addUserDialog.vue'
import { reqDeleteUser } from '@/api/userManage'
import EditUserDialog from './components/editUserDialog.vue'
import { useUserStore } from '@/stores/modules/user'

const userManageStore = useUserManageStore()
const userStore = useUserStore()

const state = reactive({
    userGroupList: [],
    addUserOpen: false,
    editUserOpen: false,
    currentUser: null as UserManage | null,
})

const userColumns = computed<TableColumnType[]>(() => { 
    return [
        {title: t('user.userName'), dataIndex: 'username', ellipsis: true},
        {title: t('user.role'), dataIndex: 'role', ellipsis: true},
        {title: t('user.userType'), dataIndex: 'authProvider'},
        {title: t('device.description'), dataIndex: 'description'},
        {title: t('device.associatedUserGroups'), dataIndex: 'userGroupList', ellipsis: true},
        {title: t('common.action'), dataIndex: 'action', width: 270},
    ]
})

const page = computed({
    get: () => userManageStore.pageLink.page,
    set: (val) => {
        userManageStore.pageLink.changePage(val)
    },
})

/** 获取用户组下拉选项 */
const getUserGroupListOptions = async () => {
    const res = await reqUserGroupListOptions()
    state.userGroupList = res.data.items
}

enum DeviceGroupActions {
    /** 编辑 */
    EDIT,
    /** 删除用户 */
    DELETE,
}

const handleAction = async (action: DeviceGroupActions, user: UserManage) => {
    switch (action) {
    case DeviceGroupActions.EDIT:
        state.currentUser = user
        state.editUserOpen = true
        break
    case DeviceGroupActions.DELETE:
        if ((userManageStore.isOnlyOneAdmin && user.role === UserRoleEnum.ADMIN) || user.isSystem || user.id === userStore.userInfo?.user?.id) return
        baseCustomModal({
            type: 'confirm',
            title: t('user.deleteUser'),
            content: t('user.deleteUserConfirmTips', { name: user.username }),
            onOk: async () => {
                await reqDeleteUser(user.id)
                await userManageStore.getUserList()
                message.success(t('common.success'))
            },
        })
        break
    }
}

/** 获取用户组选项，第二个选项及以后 */
const computedUserGroupDropDownOptions = (userGroup: {
    userGroupId: number;
    userGroupName: string;
}[]) => {
    if (userGroup.length > 2) {
        return userGroup.slice(2).map(item => {
            return new SelectOptions(item.userGroupId, item.userGroupName)
        })
    } else {
        return []
    }
}

/** 跳转用户页 */
const jumpToUserPage = (userGroupId: number) => {
    userManageStore.state.userGroupId = userGroupId
}

/** 添加用户成功回调 */
const addUserApply = () => {
    getUserGroupListOptions()
    userManageStore.getUserList()
}

/** 编辑用户成功回调 */
const editUserApply = () => {
    getUserGroupListOptions()
    userManageStore.getUserList()
}

onMounted(async () => {
    userManageStore.startPolling()
    getUserGroupListOptions()
    await userManageStore.getUserList()
})

onBeforeUnmount(() => {
    userManageStore.stopPolling()
    userManageStore.state.searchText = ''
    userManageStore.state.userGroupId = undefined
})
</script>

<style scoped lang="scss">
.user-manage-list {
  height: 100%;
  background-color: var(--gl-color-bg-surface1);
  border-radius: 10px;
  padding: 20px 24px;
  .user-manage-list-content {
    height: 100%;
    
    .user-manage-list-view {
      height: 100%;
      .header {
        height: 36px;
        margin-bottom: 16px;
        display: flex;
        justify-content: space-between;
        align-items: center;
      }
      .content {
        overflow-x: hidden;
        overflow-y: auto;
        height: calc(100% - 92px);
      }
      .pagination {
        height: 40px;
      }

      .disabled {
        cursor: not-allowed;
        color: var(--gl-color-text-disabled) !important;
      }
    }
  }
}
</style>