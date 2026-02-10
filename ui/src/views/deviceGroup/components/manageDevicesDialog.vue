<!--
 * @Author: LPY
 * @Date: 2026-02-02 09:47:46
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-05 16:30:55
 * @FilePath: \glkvm-cloud\ui\src\views\deviceGroup\components\manageDevicesDialog.vue
 * @Description: 管理设备弹窗
-->
<template>
    <BaseModal
        :width="626"
        :open="props.open"
        :title="$t('device.manageDevices')"
        destroyOnClose
        :showFooter="false"
        @close="emits('update:open', false)"
    >
        <div class="manage-device-group-container">
            <BaseInfo style="margin-bottom: 16px;">
                <div class="flex-start flex-nowrap">
                    <BaseSvg name="gl-icon-info-circle" :size="24" style="color: var(--gl-color-brand-primary); margin-right: 10px;"></BaseSvg>
                    <BaseText>{{ $t('device.manageDevicesTips') }}</BaseText>
                </div>
            </BaseInfo>
            <div class="device-table-header">
                <div>
                    <a-input-search
                        :placeholder="$t('device.searchTip')"
                        style="width: 316px; margin-right: 8px;"
                        @search="handleSearch"
                    />
                </div>

                <div v-if="hasPermission(PermissionEnum.DEVICE_GROUP_WRITE)" class="flex">
                    <BaseButton danger @click="removeDevices">{{ $t('common.remove') }}</BaseButton>
                </div>
            </div>

            <BaseLoadingContainer :spinning="state.getDeviceLoading">
                <div class="device-table">
                    <BaseTable
                        :data-source="state.deviceList"
                        :columns="deviceColumns"
                        rowKey="id"
                        :rowSelection="{ selectedRowKeys: state.selectedRowKeys, onChange: onSelectChange }"
                    >
                        <template #mac="{ record }">
                            {{ macAddressFormatter(record.mac) }}
                        </template>
                        <template #deviceGroupName="{ record }">
                            <BaseText v-if="record.deviceGroupName">{{ record.deviceGroupName }}</BaseText>
                            <BaseText variant="disabled" v-else>{{ $t('device.notAdded') }}</BaseText>
                        </template>
                    </BaseTable>
                </div>
                <div class="pagination flex-end items-end">
                    <BasePagination
                        :total="state.pageLink.total"
                        :pageSize="state.pageLink.size"
                        v-model:current="page" />
                </div>
            </BaseLoadingContainer>
        </div>
    </BaseModal>
</template>

<script setup lang="ts">
import { computed, h, reactive, watch } from 'vue'
import { BaseInfo, BaseModal } from 'gl-web-main/components'
import { baseCustomModal, macAddressFormatter, PageLink } from 'gl-web-main'
import { t } from '@/hooks/useLanguage'
import { message, TableColumnType } from 'ant-design-vue'
import { reqDeleteDevicesFromDeviceGroup } from '@/api/deviceGroup'
import BaseTable from '@/components/base/baseTable.vue'
import { DeviceInfo } from '@/models/device'
import BasePagination from '@/components/base/basePagination.vue'
import BaseLoadingContainer from '@/components/base/baseLoadingContainer.vue'
import { DeviceGroup } from '@/models/deviceGroup'
import { getDeviceListApi } from '@/api/device'
import { hasPermission } from '@/utils/permission'
import { PermissionEnum } from '@/models/permission'

const props = defineProps<{ open: boolean, currentDeviceGroup: DeviceGroup | null }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply'): void;
}>()

const state = reactive({
    searchText: '',
    pageLink: new PageLink({ size: 20 }),
    getDeviceLoading: false,
    deviceList: [] as DeviceInfo[],
    completeDeviceList: [] as DeviceInfo[],
    selectedRowKeys: [],
    selectedRows: [],
})

const handleSearch = (text: string) => {
    state.searchText = text
    state.deviceList = state.completeDeviceList.filter(d => 
        (d?.id?.toString().toLowerCase()?.indexOf(state.searchText) > -1 
                || d?.ipaddr?.indexOf(state.searchText) > -1
                || d?.mac?.indexOf(state.searchText) > -1))
}

const getDeviceList = async () => {
    state.getDeviceLoading = true
    try {
        const res = await getDeviceListApi({ groupId: props.currentDeviceGroup?.id })
        state.deviceList = res.data.items
        state.completeDeviceList = res.data.items
        state.getDeviceLoading = false
    } catch (error) {
        state.getDeviceLoading = false
    }
}

/** 下方设备列表 */
const deviceColumns = computed<TableColumnType[]>(() => { 
    return [
        {title: t('device.deviceID'), dataIndex: 'ddns', ellipsis: true},
        {title: t('device.mac'), dataIndex: 'mac', ellipsis: true},
        {title: t('device.IPAddress'), dataIndex: 'ipaddr', ellipsis: true},
    ]
})

const page = computed({
    get: () => state.pageLink.page,
    set: (val) => {
        state.pageLink.changePage(val)
    },
})

type Key = string | number;
const onSelectChange = (selectedRowKeys: Key[], selectedRows: DeviceInfo[]) => {
    state.selectedRowKeys = selectedRowKeys
    state.selectedRows = selectedRows
}

/** 移除设备 */
const removeDevices = async () => {
    if (state.selectedRows.length === 0) {
        return message.error(t('device.selectDeviceTips'))
    }

    baseCustomModal({
        type: 'confirm',
        title: t('device.removeDevice'),
        content: h('div', {}, [
            h('div', {}, t('device.currentlySelectedDevice', { num: state.selectedRows.length })),
            h('br'),
            h('div', {}, t('device.removeDeviceConfirmTips1')),
            h('div', {}, t('device.removeDeviceConfirmTips2')),
        ]),
        onOk: async () => {
            await reqDeleteDevicesFromDeviceGroup(props.currentDeviceGroup?.id, { deviceIds: state.selectedRows.map(item => item.id) })
            message.success(t('common.success'))
            getDeviceList()
            emits('handleApply')
        },
    })
}

/** 初始化数据 */
watch(() => props.open, (newVal) => {
    if (newVal) {
        init()
    }
})

const init = async () => {
    getDeviceList()
}
</script>

<style lang="scss" scoped>
.manage-device-group-container {
  .basic-info-form {
    padding: 16px;
  }

  .device-table-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .pagination {
    margin-top: 16px;
  }
}
</style>