<!--
 * @Author: LPY
 * @Date: 2026-01-29 17:57:17
 * @LastEditors: LPY
 * @LastEditTime: 2026-01-30 09:15:08
 * @FilePath: \glkvm-cloud\ui\src\views\device\components\addDeviceGroupSimpleDialog.vue
 * @Description: 创建设备组简易弹窗
-->
<template>
    <BaseModal
        :width="500"
        :open="props.open"
        :title="$t('device.addDeviceGroup')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <div>
            <AForm
                :colon="false"
                :rules="formRules"
                :model="state.formData"
                ref="formRef"
            >
                <AFormItem name="name" :label="$t('device.deviceGroupName')" :labelCol="{ span: 8 }" :wrapperCol="{ span: 16 }" labelAlign="left">
                    <AInput v-model:value="state.formData.name" :maxlength="32" :placeholder="$t('device.requiredDeviceGroupName')" style="width: 100%;" />
                </AFormItem>
                <AFormItem name="description" :label="$t('device.description')" :labelCol="{ span: 8 }" :wrapperCol="{ span: 16 }" labelAlign="left">
                    <ATextarea 
                        v-model:value="state.formData.description"
                        :maxlength="200"
                        :placeholder="$t('device.requiredDeviceGroupDescription')"
                        style="width: 100%;" />
                </AFormItem>
            </AForm>
        </div>
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import BaseModal from '@/components/base/baseModalI18n.vue'
import { FormRules, OnBeforeOk } from 'gl-web-main'
import { t } from '@/hooks/useLanguage'
import { FormInstance } from 'ant-design-vue'
import { reqAddDeviceGroup } from '@/api/device'

const props = defineProps<{ open: boolean }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply', value: number): void;
}>()

const formRef = ref<FormInstance>()

const state = reactive({
    formData: {
        name: undefined,
        description: undefined,
    },
})

/** 表单验证 */
const formRules: FormRules = {
    name: [
        { required: true, message: t('device.requiredDeviceGroupName'), trigger: 'change' },
    ],
}

/** 提交 */
const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        reqAddDeviceGroup({ name: state.formData.name, description: state.formData.description }).then((res) => {
            emits('handleApply', res.data.id)
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
    }
})

const init = async () => {
    state.formData.name = undefined
    state.formData.description = undefined
}
</script>

<style lang="scss" scoped>

</style>