<!--
 * @Author: LPY
 * @Date: 2025-08-25 10:25:14
 * @LastEditors: LPY
 * @LastEditTime: 2025-08-26 16:06:28
 * @FilePath: \glkvm-cloud\web-ui\src\views\device\components\executeCommandDialog.vue
 * @Description: 批量编辑设备弹窗。
-->
<template>
    <BaseModal
        :width="500"
        :open="props.open"
        :title="$t('device.executeCommand')"
        destroyOnClose
        class="modal"
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <AForm
            class="dense-form"
            :colon="false"
            :rules="formRules"
            :model="state.formData"
            ref="formRef"
            layout="horizontal"
            @validate="handleValidate"
        >
            <AFormItem name="username" :label="$t('device.username')">
                <div class="flex-end">
                    <AInput v-model:value="state.formData.username" name="username" :placeholder="$t('device.inputUsername')" style="width: 200px;" />
                </div>
            </AFormItem>
            <AFormItem name="cmd" :label="$t('device.command')">
                <div class="flex-end">
                    <AInput v-model:value="state.formData.cmd" name="cmd" :placeholder="$t('device.inputCommand')" style="width: 200px;" />
                </div>
            </AFormItem>
            <AFormItem name="params" :label="$t('device.parameter')">
                <div class="flex-end">
                    <ASelect 
                        v-model:value="state.formData.params"
                        name="params"
                        :placeholder="$t('device.inputParameter')"
                        mode="tags"
                        :tokenSeparators="[',']"
                        style="width: 200px;" />
                </div>
            </AFormItem>
            <AFormItem name="wait" :label="$t('device.waitTime')">
                <div class="flex-end">
                    <AInputNumber 
                        v-model:value="state.formData.wait"
                        name="wait"
                        :addonAfter="$t('common.s')"
                        :placeholder="$t('device.inputWaitTime')"
                        :min="0"
                        :max="30"
                        style="width: 200px;" />
                </div>
            </AFormItem>
        </AForm>
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import BaseModal from '@/components/base/baseModalI18n.vue'
import { FormRules, OnBeforeOk, useValidateInfo } from 'gl-web-main'
import { t } from '@/hooks/useLanguage'
import { DeviceInfo, ExecuteCommandFormData } from '@/models/device'
import { FormInstance } from 'ant-design-vue'

const props = defineProps<{ open: boolean, selection: DeviceInfo[] }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply', value: any)
}>()

const { handleValidate } = useValidateInfo()

const formRef = ref<FormInstance>()

const state = reactive<{formData: ExecuteCommandFormData}>({
    formData: {
        username: '',
        cmd: '',
        params: [],
        wait: 30,
    },
})

/** 表单验证 */
const formRules: FormRules = {
    username: [{ required: true, message: t('device.requiredUsername'), trigger: 'change' }],
    cmd: [{ required: true, message: t('device.requiredCommand'), trigger: 'change' }],
}

/** 提交 */
const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        emits('handleApply', state.formData)
        done(true)
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
    state.formData.username = ''
    state.formData.cmd = ''
    state.formData.params = []
    state.formData.wait = 30
}
</script>

<style lang="scss">
.modal {
    .base-modal-content {
        overflow: hidden !important;
    }
}
</style>