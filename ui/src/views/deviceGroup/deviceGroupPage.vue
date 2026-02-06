<!--
 * @Author: LPY
 * @Date: 2026-01-30 09:47:29
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-05 16:29:40
 * @FilePath: \glkvm-cloud\ui\src\views\deviceGroup\deviceGroupPage.vue
 * @Description: 设备组页面
-->
<template>
    <div class="device-list-container">
        <div class="out-device-list-header">
            <div class="left">
                <BaseText type="large-title-m">{{ $t('device.deviceGroup') + '(' + deviceGroupStore.deviceGroupList.length + ')' }}</BaseText>
            </div>
        </div>
        <div class="device-list">
            <div class="device-list-content">
                <BaseLoadingContainer :spinning="deviceGroupStore.state.getDeviceGroupLoading">
                    <div class="device-list-view full-height">
                        <div class="header">
                            <div>
                                <a-input-search
                                    :placeholder="$t('device.searchTip')"
                                    style="width: 316px; margin-right: 8px;"
                                    @search="deviceGroupStore.handleSearch"
                                />
                
                                <ASelect 
                                    v-model:value="deviceGroupStore.state.userGroupId"
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
                                <BaseButton 
                                    v-if="hasPermission(PermissionEnum.DEVICE_GROUP_WRITE)"
                                    type="primary"
                                    size="middle"
                                    @click="state.addDeviceGroupOpen = true">
                                    <GlSvg name="gl-icon-plus-regular" style="color: var(--gl-color-text-white);margin-right: 4px;"></GlSvg>
                                    {{ $t('device.addDeviceGroup') }}
                                </BaseButton>
                            </div>
                        </div>
                        <div class="content">
                            <BaseTable 
                                :data-source="deviceGroupStore.deviceGroupList"
                                :columns="deviceGroupColumns"
                            >
                                <template #deviceCount="{ record }">
                                    <a rel="noopener noreferrer" @click="handleAction(DeviceGroupActions.MANAGE_DEVICES, record)">{{ record.deviceCount }}</a>
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
                                        <BaseDropdownSelect :options="DEVICE_OPTIONS()" @update:value="(v) => handleAction(v, record)">
                                            <a 
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                style="margin-right: 16px;">
                                                {{ $t('common.more') }}
                                            </a>
                                        </BaseDropdownSelect>
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
                                :total="deviceGroupStore.pageLink.total"
                                :pageSize="deviceGroupStore.pageLink.size"
                                v-model:current="page" />
                        </div>
                    </div>
                </BaseLoadingContainer>
            </div> 
        </div>

        <!-- 添加设备组弹窗 -->
        <AddDeviceGroupDialog
            v-model:open="state.addDeviceGroupOpen"
            @handleApply="addDeviceGroupApply"
        />

        <!-- 管理设备弹窗 -->
        <ManageDevicesDialog
            v-model:open="state.manageDeviceOpen"
            :currentDeviceGroup="state.currentDeviceGroup"
            @handleApply="manageDevicesApply"
        />

        <!-- 添加设备到设备组弹窗 -->
        <AddDeviceToDeviceGroupDialog
            v-model:open="state.addDeviceToDeviceGroupOpen"
            :currentDeviceGroup="state.currentDeviceGroup"
            @handleApply="addDeviceToDeviceGroupApply"
        />

        <!-- 编辑设备组信息弹窗 -->
        <EditDeviceGroupInfoDialog
            v-model:open="state.editDeviceGroupInfoOpen"
            :currentDeviceGroup="state.currentDeviceGroup"
            @handleApply="editDeviceGroupInfoApply"
        />
    </div>
</template> 

<script setup lang="ts">
import BaseLoadingContainer from '@/components/base/baseLoadingContainer.vue'
import BasePagination from '@/components/base/basePagination.vue'
import BaseTable from '@/components/base/baseTable.vue'
import { t } from '@/hooks/useLanguage'
import { message, type TableColumnType } from 'ant-design-vue'
import { computed, onBeforeUnmount, onMounted, reactive } from 'vue'
import { baseCustomModal, SelectOptions } from 'gl-web-main'
import { useDeviceGroupStore } from '@/stores/modules/deviceGroup'
import { reqDeleteDeviceGroup, reqUserGroupListOptions } from '@/api/deviceGroup'
import { BaseDropdownSelect, GlSvg } from 'gl-web-main/components'
import { DeviceGroup } from '@/models/deviceGroup'
import AddDeviceGroupDialog from './components/addDeviceGroupDialog.vue'
import ManageDevicesDialog from './components/manageDevicesDialog.vue'
import AddDeviceToDeviceGroupDialog from './components/addDeviceToDeviceGroupDialog.vue'
import EditDeviceGroupInfoDialog from './components/editDeviceGroupInfoDialog.vue'
import { useUserManageStore } from '@/stores/modules/userManage'
import { useRouter } from 'vue-router'
import { hasPermission } from '@/utils/permission'
import { PermissionEnum } from '@/models/permission'

const router = useRouter()

const deviceGroupStore  = useDeviceGroupStore()
const userManageStore = useUserManageStore()

const deviceGroupColumns = computed<TableColumnType[]>(() => {
    let options: TableColumnType[] = [
        {title: t('device.deviceGroupName'), dataIndex: 'name', ellipsis: true},
        {title: t('device.associatedDeviceCount'), dataIndex: 'deviceCount', ellipsis: true},
        {title: t('device.description'), dataIndex: 'description'},
        {title: t('device.associatedUserGroups'), dataIndex: 'userGroupList', ellipsis: true},
    ]

    if (hasPermission(PermissionEnum.DEVICE_GROUP_WRITE)) {
        options.push({title: t('common.action'), dataIndex: 'action', width: 270})
    }

    return options
})

const page = computed({
    get: () => deviceGroupStore.pageLink.page,
    set: (val) => {
        deviceGroupStore.pageLink.changePage(val)
    },
})

const state = reactive({
    userGroupList: [],
    addDeviceGroupOpen: false,
    manageDeviceOpen: false,
    addDeviceToDeviceGroupOpen: false,
    editDeviceGroupInfoOpen: false,
    // 当前操作的设备组
    currentDeviceGroup: null as DeviceGroup | null,
})

enum DeviceGroupActions {
    /** 管理设备 */
    MANAGE_DEVICES,
    /** 将设备加入到设备组 */
    ADD_DEVICES_TO_GROUP,
    /** 编辑基础信息 */
    EDIT_BASIC_INFO,
    /** 删除设备组 */
    DELETE,
}

const DEVICE_OPTIONS = () => {
    return [
        new SelectOptions(DeviceGroupActions.MANAGE_DEVICES, t('device.manageDevices')),
        new SelectOptions(DeviceGroupActions.ADD_DEVICES_TO_GROUP, t('device.addDeviceToGroup')),
        new SelectOptions(DeviceGroupActions.EDIT_BASIC_INFO, t('device.editBasicInfo')),
    ]
}

const handleAction = async (action: DeviceGroupActions, deviceGroup: DeviceGroup) => {
    switch (action) {
    case DeviceGroupActions.MANAGE_DEVICES:
        state.currentDeviceGroup = deviceGroup
        state.manageDeviceOpen = true
        break
    case DeviceGroupActions.ADD_DEVICES_TO_GROUP:
        state.currentDeviceGroup = deviceGroup
        state.addDeviceToDeviceGroupOpen = true
        break
    case DeviceGroupActions.EDIT_BASIC_INFO:
        state.currentDeviceGroup = deviceGroup
        state.editDeviceGroupInfoOpen = true
        break
    case DeviceGroupActions.DELETE:
        baseCustomModal({
            type: 'confirm',
            title: t('device.deleteDeviceGroup'),
            content: t('device.deleteDeviceGroupConfirmTips'),
            okText: t('common.delete'),
            onOk: async () => {
                await reqDeleteDeviceGroup(deviceGroup.id)
                deviceGroupStore.getDeviceGroupList()
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
    router.push('/user')
}

/** 获取用户组下拉选项 */
const getUserGroupListOptions = async () => {
    const res = await reqUserGroupListOptions()
    state.userGroupList = res.data.items
}

/** 添加设备组成功回调 */
const addDeviceGroupApply = () =>{
    message.success(t('common.success'))
    deviceGroupStore.getDeviceGroupList()
}

/** 管理设备成功回调 */
const manageDevicesApply = () => {
    deviceGroupStore.getDeviceGroupList()
}

/** 添加设备到设备组成功回调 */
const addDeviceToDeviceGroupApply = () => {
    deviceGroupStore.getDeviceGroupList()
}

/** 编辑设备组信息成功回调 */
const editDeviceGroupInfoApply = () => {
    deviceGroupStore.getDeviceGroupList()
}

onMounted(async () => {
    deviceGroupStore.startPolling()
    getUserGroupListOptions()
    deviceGroupStore.getDeviceGroupList()
})

onBeforeUnmount(() => {
    deviceGroupStore.stopPolling()
    deviceGroupStore.state.searchText = ''
    deviceGroupStore.state.userGroupId = undefined
})
</script> 

<style lang="scss" scoped>
.device-list-container {
    height: 100%;
    padding: 20px 24px;
    background-color: var(--gl-color-bg-page);

    .out-device-list-header {
        height: 48px;
        margin-bottom: 16px;
        padding: 0 12px;
        display: flex;
        justify-content: space-between;
        align-items: center;

        .left {
            display: flex;
            align-items: center;
        }
    }

    .device-list {
        height: calc(100% - 64px);
        background-color: var(--gl-color-bg-surface1);
        border-radius: 10px;
        padding: 20px 24px;
        .device-list-content {
            height: 100%;
            
            .device-list-view {
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
}
</style>