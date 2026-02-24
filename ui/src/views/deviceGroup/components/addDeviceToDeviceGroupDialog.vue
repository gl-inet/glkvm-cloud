<!--
 * @Author: LPY
 * @Date: 2026-02-02 11:32:36
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 09:18:52
 * @FilePath: \glkvm-cloud\ui\src\views\deviceGroup\components\addDeviceToDeviceGroupDialog.vue
 * @Description: 将设备添加到设备组弹窗
-->
<template>
    <BaseModal
        :width="626"
        :open="props.open"
        :title="$t('device.moveToDeviceGroup')"
        destroyOnClose
        :okText="$t('common.apply')"
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <div class="add-device-to-group-container">
            <BaseInfo style="margin-bottom: 16px;">
                <div class="flex-start flex-nowrap">
                    <BaseSvg name="gl-icon-info-circle" :size="24" style="color: var(--gl-color-brand-primary); margin-right: 10px;"></BaseSvg>
                    <BaseText>{{ $t('device.moveToDeviceGroupTips') }}</BaseText>
                </div>
            </BaseInfo>

            <div class="device-table-header">
                <div>
                    <a-input-search
                        :placeholder="$t('device.searchTip')"
                        style="width: 316px; margin-right: 8px;"
                        @search="deviceStore.handleSearch"
                    />
                </div>

                <div class="flex">
                    <BaseText style="margin-right: 8px;">{{ $t('device.showOnlyUnassigned') }}</BaseText>
                    <ASwitch v-model:checked="deviceStore.state.onlyShowUnassigned" />
                </div>
            </div>

            <BaseLoadingContainer :spinning="deviceStore.state.getDeviceLoading">
                <div class="device-table">
                    <BaseTable
                        :data-source="deviceStore.deviceList"
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
                        :total="deviceStore.pageLink.total"
                        :pageSize="deviceStore.pageLink.size"
                        v-model:current="page" />
                </div>
            </BaseLoadingContainer>
        </div>
    </BaseModal>
</template>

<script setup lang="ts">
import { computed, h, reactive, watch } from 'vue'
import { BaseInfo, BaseModal } from 'gl-web-main/components'
import { baseCustomModal, macAddressFormatter, OnBeforeOk } from 'gl-web-main'
import BaseLoadingContainer from '@/components/base/baseLoadingContainer.vue'
import { useDeviceStore } from '@/stores/modules/device'
import BaseTable from '@/components/base/baseTable.vue'
import { message, TableColumnType } from 'ant-design-vue'
import { t } from '@/hooks/useLanguage'
import { DeviceInfo } from '@/models/device'
import BasePagination from '@/components/base/basePagination.vue'
import { reqMoveDevicesToDeviceGroup } from '@/api/device'
import { DeviceGroup } from '@/models/deviceGroup'

const props = defineProps<{ open: boolean, currentDeviceGroup: DeviceGroup | null }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply'): void;
}>()

const deviceStore = useDeviceStore()

const state = reactive({
    selectedRowKeys: [],
    selectedRows: [],
})

const page = computed({
    get: () => deviceStore.pageLink.page,
    set: (val) => {
        deviceStore.pageLink.changePage(val)
    },
})

/** 下方设备列表 */
const deviceColumns = computed<TableColumnType[]>(() => { 
    return [
        {title: t('device.deviceID'), dataIndex: 'ddns', ellipsis: true},
        {title: t('device.mac'), dataIndex: 'mac', ellipsis: true},
        {title: t('device.IPAddress'), dataIndex: 'ipaddr', ellipsis: true},
        {title: t('device.deviceGroup'), dataIndex: 'deviceGroupName', ellipsis: true},
    ]
})

type Key = string | number;
const onSelectChange = (selectedRowKeys: Key[], selectedRows: DeviceInfo[]) => {
    state.selectedRowKeys = selectedRowKeys
    state.selectedRows = selectedRows
}

/** 提交 */
const handleApply: OnBeforeOk = (done) => {
    if (state.selectedRows.length === 0) {
        done(false)
        return message.error(t('device.selectDeviceTips'))
    }

    baseCustomModal({
        type: 'info',
        title: t('device.addDeviceToGroup'),
        showIcon: false,
        content: h('div', {}, [
            h('div', {}, t('device.currentlySelectedDevice', { num: state.selectedRows.length })),
            h('br'),
            h('div', {}, t('device.addDeviceToGroupConfirmTips1')),
            h('div', {}, t('device.addDeviceToGroupConfirmTips2')),
        ]),
        onOk: async () => {
            try {
                await reqMoveDevicesToDeviceGroup({ deviceIds: state.selectedRows.map(d => d.id), groupId: props.currentDeviceGroup.id })
                message.success(t('common.success'))
                deviceStore.getDeviceList()
                emits('handleApply')
                done(true)
            } catch {
                done(false)
                
            }
        },
    })
}

/** 初始化数据 */
watch(() => props.open, (newVal) => {
    if (newVal) {
        init()
    } else {
        deviceStore.state.onlyShowUnassigned = false
        deviceStore.state.searchText = ''
    }
})

const init = () => {
    deviceStore.state.onlyShowUnassigned = true
    deviceStore.state.searchText = ''
    state.selectedRowKeys = []
    state.selectedRows = []
    deviceStore.getDeviceList()
}
</script>

<style lang="scss" scoped>
.add-device-to-group-container {
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