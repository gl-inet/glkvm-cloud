<!--
 * @Author: LPY
 * @Date: 2026-02-05 14:49:17
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-05 16:02:56
 * @FilePath: \glkvm-cloud\ui\src\views\device\components\accessDeviceWebDialog.vue
 * @Description: 访问用户自己的设备web弹窗
-->
<template>
    <BaseModal
        :width="500"
        :open="props.open"
        :title="$t('device.accessYourDevice')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <AForm
            :colon="false"
            :rules="formRules"
            :model="state.formData"
            :labelCol="{ span: 8 }"
            :wrapperCol="{ span: 16 }"
            ref="formRef"
        >
            <AFormItem name="protocol" :label="$t('device.protocol')" labelAlign="left">
                <ARadioGroup v-model:value="state.formData.protocol">
                    <ARadio value="http">HTTP</ARadio>
                    <ARadio value="https">HTTPS</ARadio>
                </ARadioGroup>
            </AFormItem>
            <AFormItem name="ip" :label="$t('device.IPAddress')" labelAlign="left">
                <AInput v-model:value="state.formData.ip" placeholder="127.0.0.1" style="width: 100%;" />
            </AFormItem>
            <AFormItem name="port" :label="$t('device.port')" labelAlign="left">
                <AInputNumber v-model:value="state.formData.port" placeholder="80" style="width: 100%;" />
            </AFormItem>
        </AForm>
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { BaseModal } from 'gl-web-main/components'
import { FormRules, OnBeforeOk } from 'gl-web-main'
import { t } from '@/hooks/useLanguage'
import { FormInstance } from 'ant-design-vue'
import { DeviceInfo } from '@/models/device'

const props = defineProps<{ open: boolean, currentDevice: DeviceInfo | null }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply'): void;
}>()

const formRef = ref<FormInstance>()

const state = reactive({
    formData: {
        protocol: 'http',
        ip: undefined,
        port: undefined,
    },
})

/** 表单验证 */
const formRules: FormRules = {
    ip: [{
        trigger: 'blur',
        validator: async (_: any, value: string) =>  {
            const ipv4Regex = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/
            if (!value ||ipv4Regex.test(value)) {
                return Promise.resolve()
            }
            return Promise.reject(t('device.ipNotCorrect'))
        },
    }],
    port: [{
        trigger: 'blur',
        validator: async (_: any, value: string) =>  {
            const protRegex = /^(0|[1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$/
            if (!value || protRegex.test(value)) {
                return Promise.resolve()
            }
            return Promise.reject(t('device.portNotCorrect'))
        },
    }],
}
/** 提交 */
const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        const addr = encodeURIComponent(`${state.formData.ip || '127.0.0.1'}:${state.formData.port || 80}/`)
        window.open(`/web/${props.currentDevice?.ddns}/${state.formData.protocol}/${addr}`)
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

const init = async () => {
   
}
</script>

<style lang="scss" scoped>

</style>