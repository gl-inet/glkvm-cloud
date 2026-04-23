<!--
 * @Author: LPY
 * @Date: 2026-01-30 15:05:36
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 09:28:04
 * @FilePath: \glkvm-cloud\ui\src\views\deviceGroup\components\addDeviceGroupDialog.vue
 * @Description: 添加设备组弹窗
-->
<template>
    <BaseModal
        :width="626"
        :open="props.open"
        :title="$t('device.addDeviceGroup')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <div class="add-device-group-container">
            <BaseText type="head-m">{{ $t('device.basicInfo') }}</BaseText>
            <div class="basic-info-form">
                <AForm
                    :colon="false"
                    :rules="formRules"
                    :model="state.formData"
                    :labelCol="{ span: 8 }"
                    :wrapperCol="{ span: 16 }"
                    ref="formRef"
                >
                    <AFormItem name="name" :label="$t('device.deviceGroupName')"   labelAlign="left">
                        <AInput v-model:value="state.formData.name" :maxlength="32" :placeholder="$t('device.requiredDeviceGroupName')" style="width: 100%;" />
                    </AFormItem>
                    <AFormItem name="description" :label="$t('device.description')" labelAlign="left">
                        <ATextarea 
                            v-model:value="state.formData.description"
                            :maxlength="200"
                            :placeholder="$t('device.requiredDeviceGroupDescription')"
                            style="width: 100%;" />
                    </AFormItem>
                    <AFormItem name="userGroupIds" :label="$t('device.associatedUserGroups')" labelAlign="left">
                        <ASelect v-model:value="state.formData.userGroupIds" mode="multiple" :placeholder="$t('common.pleaseSelect')" style="width: 100%;">
                            <ASelectOption v-for="item in state.userGroupList" :key="item.userGroupId" :value="item.userGroupId">{{ item.name }}</ASelectOption>
                        </ASelect>
                    </AFormItem>
                </AForm>
            </div>
            <BaseText type="head-m">{{ $t('device.addDeviceToGroup') }}</BaseText>
            <BaseInfo style="margin: 16px 0;">
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
import { computed, reactive, ref, watch } from 'vue'
import { BaseInfo } from 'gl-web-main/components'
import BaseModal from '@/components/base/baseModalI18n.vue'
import { FormRules, macAddressFormatter, OnBeforeOk } from 'gl-web-main'
import { t } from '@/hooks/useLanguage'
import { FormInstance, TableColumnType } from 'ant-design-vue'
import { reqAddDeviceGroup } from '@/api/device'
import { reqUserGroupListOptions } from '@/api/deviceGroup'
import BaseTable from '@/components/base/baseTable.vue'
import { useDeviceStore } from '@/stores/modules/device'
import { DeviceInfo } from '@/models/device'
import BasePagination from '@/components/base/basePagination.vue'
import BaseLoadingContainer from '@/components/base/baseLoadingContainer.vue'

const props = defineProps<{ open: boolean }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply'): void;
}>()

const deviceStore = useDeviceStore()

const formRef = ref<FormInstance>()

const state = reactive({
    formData: {
        name: undefined,
        description: undefined,
        userGroupIds: [],
    },
    userGroupList: [],
    selectedRowKeys: [],
    selectedRows: [],
})

/** 表单验证 */
const formRules: FormRules = {
    name: [
        { required: true, message: t('device.requiredDeviceGroupName'), trigger: 'change' },
    ],
}

/** 获取用户组下拉选项 */
const getUserGroupListOptions = async () => {
    const res = await reqUserGroupListOptions()
    state.userGroupList = res.data.items
}

/** 下方设备列表 */
const deviceColumns = computed<TableColumnType[]>(() => { 
    return [
        {title: t('device.deviceID'), dataIndex: 'ddns', ellipsis: true},
        {title: t('device.mac'), dataIndex: 'mac', ellipsis: true},
        {title: t('device.IPAddress'), dataIndex: 'ipaddr', ellipsis: true},
        {title: t('device.deviceGroup'), dataIndex: 'deviceGroupName', ellipsis: true},
    ]
})

const page = computed({
    get: () => deviceStore.pageLink.page,
    set: (val) => {
        deviceStore.pageLink.changePage(val)
    },
})

type Key = string | number;
const onSelectChange = (selectedRowKeys: Key[], selectedRows: DeviceInfo[]) => {
    state.selectedRowKeys = selectedRowKeys
    state.selectedRows = selectedRows
}

/** 提交 */
const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        reqAddDeviceGroup({ 
            name: state.formData.name,
            description: state.formData.description,
            userGroupIds: state.formData.userGroupIds,
            deviceIds: state.selectedRows.map(item => item.id), 
        }).then(() => {
            emits('handleApply')
            done(true)
        }).catch(() => {
            done(false)
        })
    }).catch(() => {
        done(false)
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

const init = async () => {
    state.formData.name = undefined
    state.formData.description = undefined
    deviceStore.state.onlyShowUnassigned = true
    deviceStore.state.searchText = ''
    state.selectedRowKeys = []
    state.selectedRows = []
    getUserGroupListOptions()
    deviceStore.getDeviceList()
}
</script>

<style lang="scss" scoped>
.add-device-group-container {
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