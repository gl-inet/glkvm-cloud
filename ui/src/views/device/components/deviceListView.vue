<!--
 * @Author: shufei.han
 * @Date: 2025-06-11 12:04:48
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-06 10:42:16
 * @FilePath: \glkvm-cloud\ui\src\views\device\components\deviceListView.vue
 * @Description: 
-->
<template>
    <BaseLoadingContainer :spinning="deviceStore.state.getDeviceLoading">
        <div class="device-list-view full-height">
            <div class="header">
                <div>
                    <a-input-search
                        :placeholder="$t('device.searchTip')"
                        style="width: 316px; margin-right: 8px;"
                        @search="deviceStore.handleSearch"
                    />
    
                    <ASelect 
                        v-model:value="deviceStore.state.deviceGroupId"
                        allowClear
                        :placeholder="$t('device.allAssociatedDeviceGroup')"
                        style="width: 224px;">
                        <ASelectOption v-for="item in state.groupList" :key="item.groupId" :value="item.groupId">{{ item.name }}</ASelectOption>
                    </ASelect>
                </div>

                <div>
                    <BaseButton size="middle" style="margin-right: 12px;" @click="refresh">{{ $t('common.refresh') }}</BaseButton>
                    <BaseButton size="middle" style="margin-right: 12px;" @click="executeCommand">{{ $t('device.executeCommand') }}</BaseButton>
                    <BaseButton 
                        v-if="hasPermission(PermissionEnum.DEVICE_GROUP_WRITE)"
                        size="middle"
                        style="margin-right: 12px;"
                        @click="moveToGroup">{{ $t('device.moveToDeviceGroup') }}</BaseButton>
                    <BaseButton v-if="hasPermission(PermissionEnum.DEVICE_WRITE)" type="primary" size="middle" @click="state.addDeviceOpen = true">
                        <GlSvg name="gl-icon-plus-regular" style="color: var(--gl-color-text-white);margin-right: 4px;"></GlSvg>
                        {{ $t('device.addDevice') }}
                    </BaseButton>
                </div>
            </div>
            <div class="content">
                <BaseTable 
                    :data-source="deviceStore.deviceList"
                    :columns="deviceColumns"
                    rowKey="id"
                    :rowSelection="{ selectedRowKeys: state.selectedRowKeys, onChange: onSelectChange }"
                >
                    <template #mac="{ record }">
                        {{ macAddressFormatter(record.mac) }}
                    </template>
                    <template #status="{ record }">
                        <BaseTag primary v-if="isDeviceOnline(record)">{{ $t('device.online') }}</BaseTag>
                        <BaseTag v-else>{{ $t('device.offline') }}</BaseTag>
                    </template>
                    <template #connectedTime="{ record }">
                        {{ record.connectedTime ? dayjs(record.connectedTime * 1000).format('YYYY-MM-DD HH:mm:ss') : '-' }}
                    </template>
                    <template #deviceGroupName="{ record }">
                        <div class="groups-a">
                            <a 
                                v-if="record.deviceGroupName"
                                rel="noopener noreferrer"
                                @click="jumpToDevicePage(record.deviceGroupId)"
                            >{{ record.deviceGroupName }}</a>
                            <BaseText variant="disabled" v-else>{{ $t('device.unassigned') }}</BaseText>
                        </div>
                    </template>
                    <template #description="{ record }">
                        <div v-ellipsis>
                            {{ record.description }}
                        </div>
                    </template>
                    <template #action="{ record }">
                        <div :class="[hasPermission(PermissionEnum.DEVICE_WRITE) ? 'flex-btw': 'flex-start']">
                            <a 
                                target="_blank"
                                rel="noopener noreferrer"
                                :class="[{'disabled': !isDeviceOnline(record)}]"
                                :style="{'margin-right': hasPermission(PermissionEnum.DEVICE_WRITE) ? '0px' : '12px'}"
                                @click="handleRemoteSSH(record.ddns, record)">{{ $t('device.remoteSSH') }}</a>
                            <a 
                                v-if="record.client === 'rtty-go'"
                                target="_blank"
                                rel="noopener noreferrer"
                                :class="[{'disabled': !isDeviceOnline(record)}]"
                                @click="handleAccessDeviceWeb(record)">{{ $t('device.remoteWeb') }}</a>
                            <a 
                                v-else
                                target="_blank"
                                rel="noopener noreferrer"
                                :class="[{'disabled': !isDeviceOnline(record)}]"
                                @click="handleRemoteControl(record.ddns, record)">
                                {{ $t('device.remoteControl') }}
                            </a>
                            <BaseDropdownSelect 
                                v-if="hasPermission(PermissionEnum.DEVICE_WRITE)"
                                :options="DEVICE_OPTIONS(record)"
                                @update:value="(v) => handleAction(v, record)">
                                <a 
                                    target="_blank"
                                    rel="noopener noreferrer">
                                    {{ $t('common.more') }}
                                </a>
                            </BaseDropdownSelect>
                        </div>
                    </template>
                </BaseTable>
            </div>
            <div class="pagination flex-end items-end">
                <BasePagination 
                    :total="deviceStore.pageLink.total"
                    :pageSize="deviceStore.pageLink.size"
                    v-model:current="page" />
            </div>
        </div>

        <!-- 添加设备弹窗 -->
        <AddDeviceDialog
            v-model:open="state.addDeviceOpen"
        />

        <!-- 批量编辑弹窗 -->
        <ExecuteCommandDialog 
            v-model:open="executeCommandOpen"
            :selection="state.selectedRows"
            @handleApply="executeCommandApply"
        />

        <!-- 执行命令结果弹窗 -->
        <CommandResponseDialog 
            v-if="commandResponseOpen"
            v-model:open="commandResponseOpen"
            :selection="state.selectedRows"
            :formData="executeCommandFormData"
        />

        <!-- 修改描述弹窗 -->
        <EditDescriptionDialog 
            v-model:open="editDescriptionOpen"
            :deviceId="editingDeviceId"
            :currentDescription="currentDescription"
            @handleApply="handleEditDescriptionApply"
        />

        <!-- 移动设备到设备组弹窗 -->
        <MoveToDeviceGroupDialog
            v-model:open="state.moveToGroupOpen"
            :selection="state.selectedRows"
            @handleApply="moveToGroupApply"
        />

        <!-- 访问设备web弹窗 -->
        <AccessDeviceWebDialog
            v-model:open="state.accessDeviceWebOpen"
            :currentDevice="state.currentDevice"
        />
    </BaseLoadingContainer>
</template> 

<script setup lang="ts">
import BaseLoadingContainer from '@/components/base/baseLoadingContainer.vue'
import BasePagination from '@/components/base/basePagination.vue'
import BaseTable from '@/components/base/baseTable.vue'
import { t } from '@/hooks/useLanguage'
import { useDeviceStore } from '@/stores/modules/device'
import { message, type TableColumnType } from 'ant-design-vue'
import { computed, reactive, ref } from 'vue'
import ExecuteCommandDialog from './executeCommandDialog.vue'
import { DeviceInfo, DeviceStatusEnum, ExecuteCommandFormData } from '@/models/device'
import CommandResponseDialog from './commandResponseDialog.vue'
import { BaseDropdownSelect, BaseTag, GlSvg } from 'gl-web-main/components'
import EditDescriptionDialog from './editDescriptionDialog.vue'
import { baseCustomModal, macAddressFormatter, SelectOptions } from 'gl-web-main'
import { reqDeleteDevice, reqDeviceGroupListOptions } from '@/api/device'
import AddDeviceDialog from './addDeviceDialog.vue'
import MoveToDeviceGroupDialog from './moveToDeviceGroupDialog.vue'
import { PermissionEnum } from '@/models/permission'
import { hasPermission } from '@/utils/permission'
import AccessDeviceWebDialog from './accessDeviceWebDialog.vue'
import dayjs from 'dayjs'

const deviceStore  = useDeviceStore()

const deviceColumns = computed<TableColumnType[]>(() => { 
    return [
        {title: t('device.deviceID'), dataIndex: 'ddns', ellipsis: true},
        {title: t('device.IPAddress'), dataIndex: 'ip', ellipsis: true},
        {title: t('device.mac'), dataIndex: 'mac', ellipsis: true},
        {title: t('device.status'), dataIndex: 'status', ellipsis: true},
        {title: t('device.connectedTime'), dataIndex: 'connectedTime', ellipsis: true},
        {title: t('user.associatedDeviceGroup'), dataIndex: 'deviceGroupName', ellipsis: true, width: 190},
        {title: t('device.description'), dataIndex: 'description'},
        {title: t('common.action'), dataIndex: 'action', width: 270},
    ]
})

const page = computed({
    get: () => deviceStore.pageLink.page,
    set: (val) => {
        deviceStore.pageLink.changePage(val)
    },
})

type Key = string | number;
const state = reactive<{
  selectedRowKeys: Key[];
  selectedRows: DeviceInfo[];
  addDeviceOpen: boolean;
  moveToGroupOpen: boolean;
  accessDeviceWebOpen: boolean;
  groupList: { groupId: number, name: string }[];
  currentDevice: DeviceInfo;
}>({
    selectedRowKeys: [], // Check here to configure the default column
    selectedRows: [],
    addDeviceOpen: false,
    moveToGroupOpen: false,
    accessDeviceWebOpen: false,
    groupList: [],
    currentDevice: null,
})

const onSelectChange = (selectedRowKeys: Key[], selectedRows: DeviceInfo[]) => {
    console.log('selectedRowKeys changed: ', selectedRowKeys)
    state.selectedRowKeys = selectedRowKeys
    state.selectedRows = selectedRows
}

/** 计算时间 */
// const calculateWithDuration = (connected: number, isMilliseconds: boolean = false) => {
//     const time = isMilliseconds ? connected / 1000 : connected
//     const day = Math.floor(time / (60 * 60 * 24))
//     const hour = Math.floor((time % (60 * 60 * 24)) / (60 * 60))
//     const minute = Math.floor((time % (60 * 60)) / (60))
//     const second = Math.floor((time % (60)))

//     return `${day}${t('common.d')} ${hour}${t('common.h')} ${minute}${t('common.m')} ${second}${t('common.s')}`
// }

/** 刷新列表 */
const refresh = () => {
    deviceStore.getDeviceList()
    message.success(t('device.refreshSuccess'))
}

/** 批量配置 */
const executeCommandOpen = ref(false)
const executeCommand = () => {
    // 判断是否选择了设备
    if (!state.selectedRowKeys.length) {
        message.error(t('device.selectDeviceTips'))
        return
    }
    executeCommandOpen.value = true
}

/** 批量配置结果弹窗 */
const commandResponseOpen = ref(false)

const executeCommandFormData = ref<ExecuteCommandFormData>()
/** 执行命令弹窗提交 */
const executeCommandApply = (formData: ExecuteCommandFormData) => {
    executeCommandFormData.value = formData
    executeCommandOpen.value = false
    commandResponseOpen.value = true
}

/** 远程SSH */
const handleRemoteSSH = async (id: string, device: DeviceInfo) => {
    if (!isDeviceOnline(device)) return
    try {
        let url = `/#/rtty/${id}`
        window.open(url)
    } catch (error) {
        console.log(error)
    }
}

/** 远程控制 */
const handleRemoteControl = async (id: string, device: DeviceInfo) => {
    if (!isDeviceOnline(device)) return
    try {
        let proto = 'https'
        let ipaddr = '127.0.0.1'
        let port = 443
        let path = '/'
        const addr = encodeURIComponent(`${ipaddr}:${port}${path}`)
        window.open(`/web/${id}/${proto}/${addr}`)
    } catch (error) {
        console.log(error)
    }
}

/** 访问设备web */
const handleAccessDeviceWeb = async (device: DeviceInfo) => {
    if (!isDeviceOnline(device)) return
    state.currentDevice = device
    state.accessDeviceWebOpen = true
}

/** 计算设备是否在线 */
const isDeviceOnline = (device: DeviceInfo) => {
    return device.status === DeviceStatusEnum.Online
}

enum DeviceActions {
    /** 编辑描述 */
    EDIT_DESCRIPTION,
    /** 删除设备 */
    DELETE,
}

const DEVICE_OPTIONS = (device: DeviceInfo) => {
    if (isDeviceOnline(device)) {
        return [
            new SelectOptions(DeviceActions.EDIT_DESCRIPTION, t('device.editDescription')),
        ]
    } else {
        return [
            new SelectOptions(DeviceActions.EDIT_DESCRIPTION, t('device.editDescription')),
            new SelectOptions(DeviceActions.DELETE, t('device.deleteDevice')),
        ]
    }
}

const handleAction = async (action: DeviceActions, device: DeviceInfo) => {
    switch (action) {
    case DeviceActions.EDIT_DESCRIPTION:
        handleEditDescription(device.id, device.description)
        break
    case DeviceActions.DELETE:
        baseCustomModal({
            type: 'confirm',
            title: t('device.deleteDevice'),
            content: t('device.deleteDeviceConfirmTips'),
            onOk: async () => {
                await reqDeleteDevice(device.id)
                deviceStore.getDeviceList()
                message.success(t('common.success'))
            },
        })
        break
    }
}

const jumpToDevicePage = (deviceGroupId: number) => {
    deviceStore.state.deviceGroupId = deviceGroupId
}

/** 修改描述 */
const editDescriptionOpen = ref(false)

const editingDeviceId = ref<number>()
const currentDescription = ref<string>('')
    
const handleEditDescription = (deviceId: number, description: string) => {
    editingDeviceId.value = deviceId
    currentDescription.value = description
    editDescriptionOpen.value = true
}
const handleEditDescriptionApply = () => {
    deviceStore.getDeviceList()
    message.success(t('common.success'))
}

/** 移动到设备组 */
const moveToGroup = () => {
    // 判断是否选择了设备
    if (!state.selectedRowKeys.length) {
        message.error(t('device.selectDeviceTips'))
        return
    }
    
    state.moveToGroupOpen = true
}

const moveToGroupApply = () => {
    state.selectedRowKeys = []
    state.selectedRows = []
    deviceStore.getDeviceList()
    message.success(t('common.success'))
}

/** 获取设备组下拉选项 */
const getDeviceGroupListOptions = async () => {
    const res = await reqDeviceGroupListOptions()
    state.groupList = res.data.items
}

getDeviceGroupListOptions()
</script> 

<style lang="scss" scoped>
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
</style>