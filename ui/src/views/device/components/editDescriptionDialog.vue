<!--
 * @Author: LPY
 * @Date: 2025-12-10 11:11:51
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 09:31:11
 * @FilePath: \glkvm-cloud\ui\src\views\device\components\editDescriptionDialog.vue
 * @Description: 修改描述弹窗
-->
<template>
    <BaseModal
        :width="500"
        :open="props.open"
        :title="$t('device.editDescription')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <AForm
            :colon="false"
            :rules="formRules"
            :model="state.formData"
            ref="formRef"
            @validate="handleValidate"
        >
            <AFormItem name="description" :label="$t('device.description')" :labelCol="{ span: 8 }" :wrapperCol="{ span: 16 }" labelAlign="left">
                <AInput v-model:value="state.formData.description" name="description" :placeholder="$t('device.inputDescription')" style="width: 100%;" />
            </AFormItem>
        </AForm>
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { BaseModal } from 'gl-web-main/components'
import { FormRules, OnBeforeOk, useValidateInfo } from 'gl-web-main'
import { t } from '@/hooks/useLanguage'
import { FormInstance } from 'ant-design-vue'
import { reqEditDescription } from '@/api/device'

const props = defineProps<{ open: boolean, deviceId: number | undefined, currentDescription: string }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply'): void;
}>()

const { handleValidate } = useValidateInfo()

const formRef = ref<FormInstance>()

const state = reactive<{formData: { description: string }}>({
    formData: {
        description: '',
    },
})

/** 表单验证 */
const formRules: FormRules = {
    description: [
        { required: true, message: t('device.requiredDescription'), trigger: 'change' },
        { max: 256, message: t('common.maxLength', { length: 256 }), trigger: 'change' },
    ],
}

/** 提交 */
const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        reqEditDescription(props.deviceId, { description: state.formData.description }).then(() => {
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
    }
})

const init = () => {
    state.formData.description = props.currentDescription || ''
}
</script>

<style lang="scss">
</style>