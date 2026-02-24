<!--
 * @Author: LPY
 * @Date: 2026-02-02 14:33:24
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 11:12:18
 * @FilePath: \glkvm-cloud\ui\src\views\userManage\userGroupManagePage.vue
 * @Description: 用户组管理页
-->
<template>
    <div class="user-group-manage-list">
        <div class="user-group-manage-list-content">
            <BaseLoadingContainer :spinning="userGroupManageStore.state.getUserGroupListLoading">
                <div class="user-group-manage-list-view full-height">
                    <div class="header">
                        <div>
                            <a-input-search
                                :placeholder="$t('device.searchTip')"
                                style="width: 316px; margin-right: 8px;"
                                @search="userGroupManageStore.handleSearch"
                            />
                
                            <ASelect 
                                v-model:value="userGroupManageStore.state.deviceGroupId"
                                allowClear
                                :placeholder="$t('device.allAssociatedDeviceGroup')"
                                style="width: 224px;">
                                <ASelectOption v-for="item in state.deviceGroupList" :key="item.groupId" :value="item.groupId">{{ item.name }}</ASelectOption>
                            </ASelect>
                        </div>
            
                        <div>
                            <BaseButton 
                                v-if="hasPermission(PermissionEnum.USER_GROUP_WRITE)"
                                type="primary"
                                size="middle"
                                @click="state.addUserGroupDialogOpen = true">
                                <GlSvg name="gl-icon-plus-regular" style="color: var(--gl-color-text-white);margin-right: 4px;"></GlSvg>
                                {{ $t('user.addUserGroup') }}
                            </BaseButton>
                        </div>
                    </div>
                    <div class="content">
                        <BaseTable
                            :data-source="userGroupManageStore.userGroupList"
                            :columns="userGroupColumns"
                        >
                            <template #deviceGroupList="{ record }">
                                <div class="groups-a">
                                    <a 
                                        v-for="item in record.deviceGroupList.slice(0, 2)"
                                        :key="item.deviceGroupId"
                                        rel="noopener noreferrer"
                                        @click="jumpToDevicePage(item.deviceGroupId)"
                                    >{{ item.deviceGroupName }}</a>

                                    <BaseDropdownSelect
                                        v-if="record.deviceGroupList?.length > 2"
                                        :options="computedDeviceGroupDropDownOptions(record.deviceGroupList)"
                                        @update:value="(v) => jumpToDevicePage(v)"
                                    >
                                        <a 
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            class="groups-a-dropdown">
                                            ...+{{ computedDeviceGroupDropDownOptions(record.deviceGroupList).length }}
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
                                    <a 
                                        rel="noopener noreferrer"
                                        style="margin-right: 16px;"
                                        @click="handleAction(DeviceGroupActions.ASSOCIATE_DEVICE_GROUP, record)">{{ $t('user.associatedDeviceGroup') }}</a>
                                    <a 
                                        rel="noopener noreferrer"
                                        style="color: var(--gl-color-error-primary);"
                                        @click="handleAction(DeviceGroupActions.DELETE, record)">{{ $t('common.delete') }}</a>
                                </div>
                            </template>
                        </BaseTable>
                    </div>
                    <div class="pagination flex-end items-end">
                        <BasePagination
                            :total="userGroupManageStore.pageLink.total"
                            :pageSize="userGroupManageStore.pageLink.size"
                            v-model:current="page" />
                    </div>
                </div>
            </BaseLoadingContainer>
        </div>

        <!-- 添加用户组弹窗 -->
        <AddUserGroupDialog
            v-model:open="state.addUserGroupDialogOpen"
            @handleApply="addUserGroupApply"
        />

        <!-- 编辑用户组弹窗 -->
        <EditUserGroupDialog
            v-model:open="state.editUserGroupDialogOpen"
            :currentUserGroup="state.currentUserGroup"
            @handleApply="editUserGroupApply"
        />

        <!-- 关联设备组弹窗 -->
        <AssociateDeviceGroupDialog
            v-model:open="state.associateDeviceGroupDialogOpen"
            :currentUserGroup="state.currentUserGroup"
            @handleApply="associateDeviceGroupApply"
        />
    </div>
</template>

<script setup lang="ts">
import BaseLoadingContainer from '@/components/base/baseLoadingContainer.vue'
import BasePagination from '@/components/base/basePagination.vue'
import BaseTable from '@/components/base/baseTable.vue'
import { t } from '@/hooks/useLanguage'
import { UserGroup } from '@/models/userManage'
import { message, TableColumnType } from 'ant-design-vue'
import { baseCustomModal, SelectOptions } from 'gl-web-main'
import { computed, h, onBeforeUnmount, onMounted, reactive } from 'vue'
import { reqDeleteUserGroup } from '@/api/userManage'
import { useUserGroupManageStore } from '@/stores/modules/userGroupManage'
import AddUserGroupDialog from './components/addUserGroupDialog.vue'
import EditUserGroupDialog from './components/editUserGroupDialog.vue'
import AssociateDeviceGroupDialog from './components/associateDeviceGroupDialog.vue'
import { BaseDropdownSelect, GlSvg } from 'gl-web-main/components'
import { useDeviceStore } from '@/stores/modules/device'
import { useRouter } from 'vue-router'
import { reqDeviceGroupListOptions } from '@/api/device'
import { hasPermission } from '@/utils/permission'
import { PermissionEnum } from '@/models/permission'

const router = useRouter()

const userGroupManageStore = useUserGroupManageStore()
const deviceStore = useDeviceStore()

const state = reactive({
    userGroupList: [],
    deviceGroupList: [],
    addUserGroupDialogOpen: false,
    editUserGroupDialogOpen: false,
    associateDeviceGroupDialogOpen: false,
    currentUserGroup: null as UserGroup | null,
})

const userGroupColumns = computed<TableColumnType[]>(() => { 
    let columns: TableColumnType[] = [
        {title: t('user.userGroupName'), dataIndex: 'userGroup', ellipsis: true},
        {title: t('device.description'), dataIndex: 'description'},
        {title: t('user.numberOfUsers'), dataIndex: 'userCount', ellipsis: true},
        {title: t('user.associatedDeviceGroups'), dataIndex: 'deviceGroupList', ellipsis: true},
    ]
    if (hasPermission(PermissionEnum.USER_GROUP_WRITE)) {
        columns = [
            ...columns,
            {title: t('common.action'), dataIndex: 'action', width: 300},
        ]
    }
    return columns
})

const page = computed({
    get: () => userGroupManageStore.pageLink.page,
    set: (val) => {
        userGroupManageStore.pageLink.changePage(val)
    },
})

/** 获取设备组下拉选项 */
const getDeviceGroupListOptions = async () => {
    const res = await reqDeviceGroupListOptions()
    state.deviceGroupList = res.data.items
}

enum DeviceGroupActions {
    /** 编辑 */
    EDIT,
    /** 关联设备组 */
    ASSOCIATE_DEVICE_GROUP,
    /** 删除用户组 */
    DELETE,
}

const handleAction = async (action: DeviceGroupActions, userGroup: UserGroup) => {
    switch (action) {
    case DeviceGroupActions.EDIT:
        state.currentUserGroup = userGroup
        state.editUserGroupDialogOpen = true
        break
    case DeviceGroupActions.ASSOCIATE_DEVICE_GROUP:
        state.currentUserGroup = userGroup
        state.associateDeviceGroupDialogOpen = true
        break
    case DeviceGroupActions.DELETE:
        baseCustomModal({
            type: 'confirm',
            title: t('user.deleteUser'),
            content: h('div', {}, [
                h('div', {}, t('user.deleteUserGroupConfirmTips1')),
                h('div', {}, t('user.deleteUserGroupConfirmTips2')),
                h('div', {}, t('user.deleteUserGroupConfirmTips3')),
            ]),
            onOk: async () => {
                await reqDeleteUserGroup(userGroup.id)
                await userGroupManageStore.getUserGroupList()
                message.success(t('common.success'))
            },
        })
        break
    }
}

/** 获取设备组选项，第二个选项及以后 */
const computedDeviceGroupDropDownOptions = (deviceGroup: {
    deviceGroupId: number;
    deviceGroupName: string;
}[]) => {
    if (deviceGroup.length > 2) {
        return deviceGroup.slice(2).map(item => {
            return new SelectOptions(item.deviceGroupId, item.deviceGroupName)
        })
    } else {
        return []
    }
}

/** 跳转用户页 */
const jumpToDevicePage = (deviceGroupId: number) => {
    deviceStore.state.deviceGroupId = deviceGroupId
    router.push('/device')
}

/** 添加用户成功回调 */
const addUserGroupApply = () => {
    userGroupManageStore.getUserGroupList()
}

/** 编辑用户成功回调 */
const editUserGroupApply = () => {
    userGroupManageStore.getUserGroupList()
}

/** 关联设备组成功回调 */
const associateDeviceGroupApply = () => {
    userGroupManageStore.getUserGroupList()
}

onMounted(async () => {
    userGroupManageStore.startPolling()
    getDeviceGroupListOptions()
    await userGroupManageStore.getUserGroupList()
})

onBeforeUnmount(() => {
    userGroupManageStore.stopPolling()
    userGroupManageStore.state.searchText = ''
    userGroupManageStore.state.deviceGroupId = undefined
})
</script>

<style scoped lang="scss">
.user-group-manage-list {
  height: 100%;
  background-color: var(--gl-color-bg-surface1);
  border-radius: 10px;
  padding: 20px 24px;
  .user-group-manage-list-content {
    height: 100%;
    
    .user-group-manage-list-view {
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
        color: var(--gl-color-text-disabled);
      }
    }
  }
}
</style>