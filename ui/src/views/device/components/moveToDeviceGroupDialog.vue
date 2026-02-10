<!--
 * @Author: LPY
 * @Date: 2026-01-29 16:40:35
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-03 15:57:18
 * @FilePath: \glkvm-cloud\ui\src\views\device\components\moveToDeviceGroupDialog.vue
 * @Description: 移动到设备组弹窗
-->
<template>
    <BaseModal
        :width="500"
        :open="props.open"
        :title="$t('device.moveToDeviceGroup')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <div>
            <BaseInfo style="margin-bottom: 32px;">
                <div class="flex-start flex-nowrap">
                    <BaseSvg name="gl-icon-info-circle" :size="24" style="color: var(--gl-color-brand-primary); margin-right: 10px;"></BaseSvg>
                    <BaseText>{{ $t('device.moveToDeviceGroupTips') }}</BaseText>
                </div>
            </BaseInfo>

            <AForm
                :colon="false"
                :rules="formRules"
                :model="state.formData"
                ref="formRef"
            >
                <div class="flex-end">
                    <a @click="state.AddDeviceGroupSimpleDialogOpen = true">{{ $t('device.notFoundDeviceGroup') }}</a>
                </div>
                <AFormItem name="groupId" :label="$t('device.moveToGroup')" :labelCol="{ span: 8 }" :wrapperCol="{ span: 16 }" labelAlign="left">
                    <ASelect v-model:value="state.formData.groupId" name="groupId" :placeholder="$t('common.pleaseSelect')" style="width: 100%;">
                        <ASelectOption v-for="item in state.groupList" :key="item.groupId" :value="item.groupId">{{ item.name }}</ASelectOption>
                    </ASelect>
                </AFormItem>
            </AForm>
        </div>

        <!-- 添加设备组弹窗 -->
        <AddDeviceGroupSimpleDialog
            v-model:open="state.AddDeviceGroupSimpleDialogOpen"
            @handleApply="addDeviceGroupApply"
        />
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { BaseInfo, BaseModal } from 'gl-web-main/components'
import { FormRules, OnBeforeOk } from 'gl-web-main'
import { t } from '@/hooks/useLanguage'
import { FormInstance } from 'ant-design-vue'
import { reqDeviceGroupListOptions, reqMoveDevicesToDeviceGroup } from '@/api/device'
import { DeviceInfo } from '@/models/device'
import AddDeviceGroupSimpleDialog from './addDeviceGroupSimpleDialog.vue'

const props = defineProps<{ open: boolean, selection: DeviceInfo[] }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply'): void;
}>()

const formRef = ref<FormInstance>()

const state = reactive({
    groupList: [],
    formData: {
        groupId: undefined,
    },
    AddDeviceGroupSimpleDialogOpen: false,
})

/** 表单验证 */
const formRules: FormRules = {
    groupId: [
        { required: true, message: t('device.requiredDeviceGroup'), trigger: 'change' },
    ],
}

const getDeviceGroupListOptions = async () => {
    const res = await reqDeviceGroupListOptions()
    state.groupList = res.data.items
}

/** 提交 */
const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        reqMoveDevicesToDeviceGroup({ deviceIds: props.selection.map(d => d.id), groupId: state.formData.groupId }).then(() => {
            emits('handleApply')
            done(true)
        }).catch(() => {
            done(false)
        })
    }).catch(() => {
        done(false)
    })
}

/** 添加设备组弹窗完成 */
const addDeviceGroupApply = async (id: number) => {
    await getDeviceGroupListOptions()
    state.formData.groupId = id
}

/** 初始化数据 */
watch(() => props.open, (newVal) => {
    if (newVal) {
        init()
    }
})

const init = () => {
    getDeviceGroupListOptions()
}
</script>

<style lang="scss" scoped>

</style>