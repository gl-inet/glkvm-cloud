<!--
 * @Description: 启用 2FA 弹窗（生成 secret + 扫码 + 验证）
-->
<template>
    <BaseModal
        :width="560"
        :open="props.open"
        :title="$t('personalCenter.enable2fa')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <BaseLoadingContainer :spinning="state.loading">
            <div class="enable-2fa">
                <div class="enable-2fa-step">
                    <div class="enable-2fa-step-title">
                        1. {{ $t('personalCenter.scanQrCode') }}
                    </div>
                    <div class="enable-2fa-qr">
                        <img v-if="state.qrDataUrl" :src="state.qrDataUrl" :alt="$t('personalCenter.scanQrCode')" />
                    </div>
                    <div class="enable-2fa-secret">
                        <div class="enable-2fa-secret-label">{{ $t('personalCenter.secretKey') }}</div>
                        <div class="enable-2fa-secret-value">
                            <code>{{ state.secret || '--' }}</code>
                        </div>
                    </div>
                </div>

                <div class="enable-2fa-step">
                    <div class="enable-2fa-step-title">
                        2. {{ $t('personalCenter.enterVerifyCode') }}
                    </div>
                    <AForm
                        ref="formRef"
                        :colon="false"
                        :model="state.formData"
                        :rules="formRules"
                    >
                        <AFormItem name="code">
                            <AInput
                                v-model:value="state.formData.code"
                                :placeholder="$t('personalCenter.enterVerifyCode')"
                                :maxlength="6"
                                style="width: 100%;"
                            />
                        </AFormItem>
                    </AForm>
                </div>
            </div>
        </BaseLoadingContainer>
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { BaseModal } from 'gl-web-main/components'
import { FormRules, OnBeforeOk } from 'gl-web-main'
import { FormInstance } from 'ant-design-vue'
import QRCode from 'qrcode'
import BaseLoadingContainer from '@/components/base/baseLoadingContainer.vue'
import { reqEnable2fa, reqSetup2fa } from '@/api/personal'
import { t } from '@/hooks/useLanguage'

const props = defineProps<{ open: boolean, username?: string }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void
    (e: 'handleApply'): void
}>()

const formRef = ref<FormInstance>()

const state = reactive({
    loading: false,
    secret: '',
    otpauthUrl: '',
    qrDataUrl: '',
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

const setup = async () => {
    state.loading = true
    state.secret = ''
    state.otpauthUrl = ''
    state.qrDataUrl = ''
    state.formData.code = ''
    try {
        const res = await reqSetup2fa()
        state.secret = res.data.secret
        state.otpauthUrl = res.data.otpauthUrl
        state.qrDataUrl = await QRCode.toDataURL(state.otpauthUrl, { margin: 1, width: 200 })
    } finally {
        state.loading = false
    }
}

watch(() => props.open, (open) => {
    if (open) setup()
})

const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        reqEnable2fa({ secret: state.secret, code: state.formData.code })
            .then(() => {
                emits('handleApply')
                done(true)
            })
            .catch(() => done(false))
    }).catch(() => done(false))
}
</script>

<style scoped lang="scss">
.enable-2fa {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.enable-2fa-step-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--gl-color-text-level1);
  margin-bottom: 12px;
}

.enable-2fa-qr {
  display: flex;
  justify-content: center;
  padding: 12px 0;

  img {
    width: 200px;
    height: 200px;
    background-color: #fff;
    border: 1px solid var(--gl-color-line-divider2);
    border-radius: 4px;
  }
}

.enable-2fa-secret {
  text-align: center;
}

.enable-2fa-secret-label {
  font-size: 12px;
  color: var(--gl-color-text-level3);
  margin-bottom: 4px;
}

.enable-2fa-secret-value code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, monospace;
  font-size: 13px;
  letter-spacing: 1px;
  color: var(--gl-color-text-level1);
  word-break: break-all;
}
</style>
