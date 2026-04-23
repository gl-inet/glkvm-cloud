<!--
 * @Description: 关闭 2FA 弹窗（验证 TOTP 后清空 secret 与所有信任设备）
-->
<template>
    <BaseModal
        :width="480"
        :open="props.open"
        :title="$t('personalCenter.disable2fa')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <div class="disable-2fa">
            <div class="disable-2fa-tip">{{ $t('personalCenter.disable2faTip') }}</div>
            <AForm
                ref="formRef"
                :colon="false"
                :model="state.formData"
                :rules="formRules"
            >
                <AFormItem name="code" :label="$t('personalCenter.verifyCode')" labelAlign="left">
                    <AInput
                        v-model:value="state.formData.code"
                        :placeholder="$t('personalCenter.enterVerifyCode')"
                        :maxlength="6"
                        style="width: 100%;"
                    />
                </AFormItem>
            </AForm>
        </div>
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import BaseModal from '@/components/base/baseModalI18n.vue'
import { FormRules, OnBeforeOk } from 'gl-web-main'
import { FormInstance } from 'ant-design-vue'
import { reqDisable2fa } from '@/api/personal'
import { t } from '@/hooks/useLanguage'

const props = defineProps<{ open: boolean }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void
    (e: 'handleApply'): void
}>()

const formRef = ref<FormInstance>()

const state = reactive({
    formData: {
        code: '',
    },
})

const formRules: FormRules = {
    code: [
        { required: true, message: t('personalCenter.enterVerifyCode'), trigger: 'change' },
        { len: 6, message: t('personalCenter.enterVerifyCode'), trigger: 'change' },
    ],
}

watch(() => props.open, (open) => {
    if (open) state.formData.code = ''
})

const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        reqDisable2fa({ code: state.formData.code })
            .then(() => {
                emits('handleApply')
                done(true)
            })
            .catch(() => done(false))
    }).catch(() => done(false))
}
</script>

<style scoped lang="scss">
.disable-2fa {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.disable-2fa-tip {
  font-size: 13px;
  color: var(--gl-color-text-level2);
}
</style>
