<!--
 * @Author: LPY
 * @Date: 2025-08-25 09:32:42
 * @LastEditors: LPY
 * @LastEditTime: 2026-03-25 10:11:49
 * @FilePath: \glkvm-cloud\ui\src\views\device\components\addDeviceDialog.vue
 * @Description: 添加设备弹窗
-->
<template>
    <BaseModal
        :width="560"
        :open="props.open"
        :title="$t('device.addDevice')"
        destroyOnClose
        :showCancel="false"
        :okText="$t('device.copyAccessCode')"
        @ok="handleCopy"
        @close="emits('update:open', false)"
    >
        <div class="top-tips">
            <div class="tips-icon">
                <BaseSvg name="gl-icon-device" :size="16"></BaseSvg>
            </div>
            <BaseText>{{ $t('device.accessCodeTip') }}</BaseText>
        </div>

        <div class="generate-row">
            <BaseText>{{ $t('device.accessCodeCount') }}</BaseText>
            <AInputNumber v-model:value="state.count" :min="1" :max="MAX_CODES" :precision="0" />
            <BaseButton type="primary" :loading="state.generating" @click="handleGenerate">
                {{ $t('device.generateAccessCode') }}
            </BaseButton>
        </div>
        <template v-if="state.generated.length">
            <BaseInfo warning style="font-size: 12px;">{{ $t('device.accessCodeShownOnce') }}</BaseInfo>
            <pre class="script-box code-box"><BaseText type="body-r">{{ generatedText }}</BaseText></pre>
        </template>

        <BaseText type="body-m" class="codes-title">{{ $t('device.recentAccessCodes') }}</BaseText>
        <div class="codes-list">
            <BaseText v-if="!state.codes.length" type="caption-r" class="codes-empty">{{ $t('device.noAccessCodes') }}</BaseText>
            <div v-for="item in state.codes" :key="item.id" class="code-row">
                <BaseText type="caption-r" class="code-time">{{ formatTime(item.createdAt) }}</BaseText>
                <BaseTag :primary="item.status === 'unused'">{{ $t(`device.accessCodeStatus.${item.status}`) }}</BaseTag>
                <BaseText type="caption-r" class="code-device">{{ item.devid || '' }}</BaseText>
                <APopconfirm
                    v-if="item.status === 'unused'"
                    :title="$t('device.revokeAccessCodeConfirm')"
                    @confirm="handleRevoke(item.id)"
                >
                    <BaseButton danger size="small">{{ $t('device.revokeAccessCode') }}</BaseButton>
                </APopconfirm>
            </div>
        </div>
    </BaseModal>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { BaseInfo } from 'gl-web-main/components'
import BaseModal from '@/components/base/baseModalI18n.vue'
import { createRegistrationCodesApi, listRegistrationCodesApi, revokeRegistrationCodeApi } from '@/api/device'
import { copyText } from 'gl-web-main'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import { t } from '@/hooks/useLanguage'
import type { CreatedRegistrationCode, RegistrationCode } from '@/models/device'

/** 一次最多生成的接入码数（与后端一致） */
const MAX_CODES = 20

const props = defineProps<{ open: boolean }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
}>()

const state = reactive({
    count: 1,
    generating: false,
    generated: [] as CreatedRegistrationCode[],
    codes: [] as RegistrationCode[],
})

const generatedText = computed(() => state.generated.map(item => item.accessCode).join('\n'))

const formatTime = (ts: number) => dayjs.unix(ts).format('YYYY-MM-DD HH:mm')

/** 生成接入码 */
const handleGenerate = async () => {
    state.generating = true
    try {
        const res = await createRegistrationCodesApi(state.count)
        state.generated = res.data.items
        await loadCodes()
    } finally {
        state.generating = false
    }
}

/** 作废接入码 */
const handleRevoke = async (id: number) => {
    await revokeRegistrationCodeApi(id)
    message.success(t('device.revokeAccessCodeSuccess'))
    await loadCodes()
}

/** 复制新生成的接入码 */
const handleCopy = () => {
    const text = generatedText.value
    if (!text) {
        message.warning(t('device.generateAccessCodeFirst'))
        return
    }
    try {
        copyText(text)
        message.success(t('common.copySuccess'))
    } catch {
        message.error(t('common.copyFailed'))
    }
}

const loadCodes = async () => {
    const res = await listRegistrationCodesApi()
    state.codes = res.data.items
}

/** 初始化数据 */
watch(() => props.open, (newVal) => {
    if (newVal) {
        init()
    }
})

const init = () => {
    state.generated = []
    state.count = 1
    loadCodes()
}
</script>

<style lang="scss" scoped>
.script-box {
  border-radius: 6px;
  background: var(--gl-color-bg-surface2);
  padding: 10px;
  overflow: auto;
  max-height: 375px;
  font-family: unset;
}

.top-tips {
  display: flex;
  align-items: center;
  padding: 8px 10px;
  border: 1px solid var(--gl-color-brand-primary);
  background-color: var(--gl-color-brand-background);
  border-radius: 6px;

  .tips-icon {
    display: flex;
    align-items: center;
    margin-right: 10px;
    color: var(--gl-color-brand-primary);
  }
}

.generate-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.code-box {
  margin: 8px 0 0;
  max-height: 160px;
  white-space: pre-wrap;
  word-break: break-all;
}

.codes-title {
  display: block;
  margin: 16px 0 8px;
}

.codes-list {
  max-height: 220px;
  overflow: auto;
}

.codes-empty {
  color: var(--gl-color-text-secondary);
}

.code-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 32px;
  border-bottom: 1px solid var(--gl-color-bg-surface2);

  .code-time {
    width: 120px;
    flex-shrink: 0;
  }

  .code-device {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>