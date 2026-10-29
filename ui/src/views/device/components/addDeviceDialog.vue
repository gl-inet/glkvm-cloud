<!--
 * @Author: LPY
 * @Date: 2025-08-25 09:32:42
 * @LastEditors: LPY
 * @LastEditTime: 2026-03-25 10:11:49
 * @FilePath: \glkvm-cloud\ui\src\views\device\components\addDeviceDialog.vue
 * @Description: 添加设备弹窗：生成接入码 → 在设备上粘贴 → 等待设备上线
-->
<template>
    <BaseModal
        :width="600"
        :open="props.open"
        :title="$t('device.addDevice')"
        destroyOnClose
        :showCancel="false"
        :okText="$t('device.addDeviceDone')"
        @ok="close"
        @close="close"
    >
        <BaseText el="div" type="body-r" class="subtitle">{{ $t('device.addDeviceSubtitle') }}</BaseText>
        <div class="prereq">
            <BaseSvg name="gl-icon-info-circle" :size="16" class="prereq-icon"></BaseSvg>
            <BaseText type="body-r">{{ $t('device.addDeviceFirmwareTip', { version: MIN_FIRMWARE }) }}</BaseText>
        </div>

        <ol class="steps">
            <AddDeviceStep :index="1" :state="step1" :title="$t('device.stepGenerate')">
                <template v-if="!state.codes.length">
                    <BaseText el="div" type="body-r" class="step-desc">{{ $t('device.stepGenerateDesc') }}</BaseText>
                    <div v-if="!state.batchOpen" class="actions">
                        <BaseButton type="primary" :loading="state.generating" @click="generate(1)">
                            {{ $t('device.generateAccessCode') }}
                        </BaseButton>
                        <button type="button" class="link" @click="state.batchOpen = true">{{ $t('device.batchGenerateLink') }}</button>
                    </div>
                </template>

                <template v-else>
                    <div v-if="isBatch" class="codes-head">
                        <BaseText type="body-r">{{ $t('device.generatedCount', { n: state.codes.length }) }}</BaseText>
                        <button type="button" class="link" @click="copyAll">
                            {{ state.copied === 'all' ? $t('device.copiedAll') : $t('device.copyAll') }}
                        </button>
                    </div>
                    <div class="code-list">
                        <div
                            v-for="(item, index) in state.codes"
                            :key="item.id"
                            class="code-card"
                            :class="{ settled: item.status !== 'unused' }"
                        >
                            <span v-if="isBatch" class="code-no">#{{ index + 1 }}</span>
                            <code class="code-text" :title="item.accessCode">{{ shorten(item.accessCode) }}</code>
                            <span v-if="item.status === 'used'" class="code-chip joined">
                                <BaseSvg name="gl-icon-check-regular" :size="12"></BaseSvg>{{ item.devid }}
                            </span>
                            <span v-else-if="item.status !== 'unused'" class="code-chip">
                                {{ $t(`device.accessCodeStatus.${item.status}`) }}
                            </span>
                            <BaseButton v-else size="small" class="copy-btn" @click="copyOne(item)">
                                <svg v-if="state.copied === item.id" class="icon" viewBox="0 0 24 24" aria-hidden="true">
                                    <path d="M5 12.5l4.5 4.5L19 7.5"></path>
                                </svg>
                                <svg v-else class="icon" viewBox="0 0 24 24" aria-hidden="true">
                                    <rect x="9" y="9" width="11" height="11" rx="2"></rect>
                                    <path d="M5 15V5a2 2 0 0 1 2-2h10"></path>
                                </svg>
                                {{ state.copied === item.id ? $t('device.copied') : $t('device.copy') }}
                            </BaseButton>
                        </div>
                    </div>
                    <div class="code-warn">
                        <svg class="icon" viewBox="0 0 24 24" aria-hidden="true">
                            <circle cx="12" cy="12" r="9"></circle>
                            <path d="M12 7v5l3 2"></path>
                        </svg>
                        <span>{{ $t('device.accessCodeShownOnce') }}</span>
                    </div>
                    <div v-if="!state.batchOpen" class="more">
                        <span>{{ $t('device.addMoreDevices') }}</span>
                        <button type="button" class="link" :disabled="state.generating" @click="generate(1)">{{ $t('device.generateOneMore') }}</button>
                        <span aria-hidden="true">·</span>
                        <button type="button" class="link" @click="state.batchOpen = true">{{ $t('device.batchGenerate') }}</button>
                    </div>
                </template>

                <div v-if="state.batchOpen" class="batch-panel">
                    <BaseText type="body-r">{{ $t('device.accessCodeCount') }}</BaseText>
                    <AInputNumber v-model:value="state.count" :min="2" :max="MAX_CODES" :precision="0" class="count-input" />
                    <BaseText type="caption-r" class="muted">{{ $t('device.maxCodesHint', { n: MAX_CODES }) }}</BaseText>
                    <span class="spacer"></span>
                    <button type="button" class="link" @click="state.batchOpen = false">{{ $t('common.cancel') }}</button>
                    <BaseButton type="primary" :loading="state.generating" @click="generate(batchCount)">
                        {{ $t('device.generateN', { n: batchCount }) }}
                    </BaseButton>
                </div>
            </AddDeviceStep>

            <AddDeviceStep :index="2" :state="step2" :title="$t('device.stepPaste')">
                <BaseText el="div" type="body-r" class="step-desc">{{ $t('device.stepPasteDesc') }}</BaseText>
                <!-- 示意：设备本地网页上粘贴接入码的位置 -->
                <div class="sketch" aria-hidden="true">
                    <div class="sketch-head">
                        <span>{{ $t('device.pasteSketchPath') }}</span>
                        <span>{{ $t('device.pasteSketchTag') }}</span>
                    </div>
                    <div class="sketch-row">
                        <div class="sketch-input">{{ $t('device.pasteSketchPlaceholder') }}</div>
                        <div class="sketch-btn">{{ $t('device.pasteSketchSave') }}</div>
                    </div>
                </div>
            </AddDeviceStep>

            <AddDeviceStep :index="3" :state="step2" :title="$t('device.stepOnline')" last>
                <BaseText v-if="!state.codes.length" el="div" type="body-r" class="step-desc muted">
                    {{ $t('device.stepOnlineIdle') }}
                </BaseText>

                <template v-else-if="!isBatch">
                    <div v-if="state.codes[0].status === 'unused'" class="banner banner-wait">
                        <svg class="icon spin" viewBox="0 0 24 24" aria-hidden="true"><path d="M21 12a9 9 0 1 1-6.2-8.56"></path></svg>
                        <span>{{ $t('device.waitingDevice') }}</span>
                    </div>
                    <div v-else-if="state.codes[0].status === 'used'" class="banner banner-done">
                        <BaseSvg name="gl-icon-circle-check-solid" :size="16"></BaseSvg>
                        <span class="banner-text">{{ $t('device.deviceJoined', { devid: state.codes[0].devid }) }}</span>
                        <button type="button" class="link" @click="viewDevices">{{ $t('device.viewDevices') }}</button>
                    </div>
                    <div v-else class="banner banner-neutral">
                        <span>{{ state.codes[0].status === 'revoked' ? $t('device.codeRevokedNote') : $t('device.codeExpiredNote') }}</span>
                    </div>
                </template>

                <div v-else class="progress-box">
                    <div class="progress-head">
                        <svg v-if="waiting" class="icon spin brand" viewBox="0 0 24 24" aria-hidden="true"><path d="M21 12a9 9 0 1 1-6.2-8.56"></path></svg>
                        <BaseSvg v-else-if="allJoined" name="gl-icon-circle-check-solid" :size="16" class="success"></BaseSvg>
                        <span class="banner-text">{{ $t('device.joinedProgress', { n: joinedCount, total: state.codes.length }) }}</span>
                        <button v-if="joinedCount" type="button" class="link" @click="viewDevices">{{ $t('device.viewDevices') }}</button>
                    </div>
                    <div class="progress-track">
                        <div class="progress-bar" :style="{ width: `${progress}%` }"></div>
                    </div>
                </div>
            </AddDeviceStep>
        </ol>

        <AccessCodeHistory :items="state.history" @revoke="revoke" />
    </BaseModal>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, watch } from 'vue'
import BaseModal from '@/components/base/baseModalI18n.vue'
import AddDeviceStep from './addDeviceStep.vue'
import AccessCodeHistory from './accessCodeHistory.vue'
import { createRegistrationCodesApi, listRegistrationCodesApi, revokeRegistrationCodeApi } from '@/api/device'
import { baseCustomModal, copyText } from 'gl-web-main'
import { message } from 'ant-design-vue'
import { t } from '@/hooks/useLanguage'
import type { RegistrationCode, RegistrationCodeStatus } from '@/models/device'
import { useDeviceStore } from '@/stores/modules/device'

/** 支持接入码接入的最低固件版本 */
const MIN_FIRMWARE = 'V1.11.0'
/** 一次最多生成的接入码数（与后端一致） */
const MAX_CODES = 5
/** 等待设备接入时查询的间隔 */
const POLL_INTERVAL = 3000
/** 一次查询最多带的接入码数（与后端一致） */
const POLL_MAX_IDS = 50

/** 这次弹窗里生成的接入码：明文只在这里出现 */
interface GeneratedCode {
    id: number
    accessCode: string
    status: RegistrationCodeStatus
    devid: string
}

type StepState = 'todo' | 'active' | 'done'

const props = defineProps<{ open: boolean }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
}>()

const deviceStore = useDeviceStore()

const state = reactive({
    count: 3 as number | null,
    generating: false,
    batchOpen: false,
    codes: [] as GeneratedCode[],
    /** 刚复制的接入码 id，'all' 表示复制了全部 */
    copied: 0 as number | 'all',
    history: [] as RegistrationCode[],
})

/** 输入框清空时按最小值算 */
const batchCount = computed(() => Math.min(Math.max(state.count || 2, 2), MAX_CODES))

const isBatch = computed(() => state.codes.length > 1)
const joinedCount = computed(() => state.codes.filter(item => item.status === 'used').length)
const allJoined = computed(() => state.codes.length > 0 && joinedCount.value === state.codes.length)
const waiting = computed(() => state.codes.some(item => item.status === 'unused'))
const progress = computed(() => (state.codes.length ? Math.round(joinedCount.value * 100 / state.codes.length) : 0))

/** 第一步生成了就算完成；第二、三步等设备全部接入才算完成 */
const step1 = computed<StepState>(() => (state.codes.length ? 'done' : 'active'))
const step2 = computed<StepState>(() => {
    if (allJoined.value) return 'done'
    return state.codes.length ? 'active' : 'todo'
})

/** 接入码很长：只显示首尾，完整内容在复制和悬停提示里 */
const shorten = (code: string) => (code.length > 40 ? `${code.slice(0, 26)}…${code.slice(-10)}` : code)

let copiedTimer: number | undefined
let pollTimer: number | undefined
let polling = false
/** 复制过的接入码：没复制过的关掉弹窗后就再也拿不到了 */
const copiedIds = new Set<number>()

/** 作废这些接入码，失败的留着等 24 小时过期 */
const revokeCodes = (ids: number[]) => Promise.allSettled(ids.map(id => revokeRegistrationCodeApi(id)))

/** 用查到的状态更新本次生成的接入码，返回是否有设备新接入 */
const sync = (items: RegistrationCode[]) => {
    let joined = false
    for (const rc of items) {
        const item = state.codes.find(code => code.id === rc.id)
        if (!item || item.status !== 'unused' || rc.status === 'unused') continue
        item.status = rc.status
        item.devid = rc.devid || ''
        joined = joined || rc.status === 'used'
    }
    return joined
}

const loadHistory = async () => {
    const res = await listRegistrationCodesApi()
    state.history = res.data.items
    sync(state.history)
}

const stopPolling = () => {
    window.clearInterval(pollTimer)
    pollTimer = undefined
}

/** 等待设备接入：只查本次生成、还没用掉的接入码 */
const poll = async () => {
    const ids = state.codes.filter(item => item.status === 'unused').map(item => item.id).slice(-POLL_MAX_IDS)
    if (!ids.length) {
        stopPolling()
        return
    }
    if (polling) return
    polling = true
    try {
        const res = await listRegistrationCodesApi(ids)
        if (sync(res.data.items)) {
            deviceStore.getDeviceList()
            loadHistory()
        }
    } finally {
        polling = false
    }
}

const startPolling = () => {
    if (!pollTimer) pollTimer = window.setInterval(poll, POLL_INTERVAL)
}

/** 生成接入码 */
const generate = async (count: number) => {
    state.generating = true
    try {
        const res = await createRegistrationCodesApi(count)
        // 请求途中弹窗已经关了：这些码谁也看不到，直接作废
        if (!props.open) {
            revokeCodes(res.data.items.map(item => item.id))
            return
        }
        state.codes.push(...res.data.items.map(item => ({
            id: item.id, accessCode: item.accessCode, status: 'unused' as RegistrationCodeStatus, devid: '',
        })))
        state.batchOpen = false
        startPolling()
        await loadHistory()
    } finally {
        state.generating = false
    }
}

const copy = async (text: string, key: number | 'all', ids: number[]) => {
    try {
        await copyText(text)
    } catch {
        message.error(t('common.copyFailed'))
        return
    }
    ids.forEach(id => copiedIds.add(id))
    state.copied = key
    window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => {
        state.copied = 0
    }, 1600)
}
const copyOne = (item: GeneratedCode) => copy(item.accessCode, item.id, [item.id])
const copyAll = () => copy(state.codes.map(item => item.accessCode).join('\n'), 'all', state.codes.map(item => item.id))

/** 作废接入码 */
const revoke = async (id: number) => {
    await revokeRegistrationCodeApi(id)
    message.success(t('device.revokeAccessCodeSuccess'))
    await loadHistory()
}

/** 关闭：还有没复制、也没用掉的接入码时先确认，这些码关掉后明文就没了，一并作废 */
const close = () => {
    const uncopied = state.codes.filter(item => item.status === 'unused' && !copiedIds.has(item.id))
    if (!uncopied.length) {
        emits('update:open', false)
        return
    }
    baseCustomModal({
        type: 'confirm',
        title: t('device.uncopiedCodesTitle'),
        content: t('device.uncopiedCodesContent', { n: uncopied.length }, uncopied.length),
        okText: t('device.revokeAndClose'),
        cancelText: t('device.backToCopy'),
        onOk: async () => {
            await revokeCodes(uncopied.map(item => item.id))
            emits('update:open', false)
        },
    })
}

const viewDevices = () => {
    deviceStore.getDeviceList()
    close()
}

/** 打开时重置：接入码明文只在这一次弹窗里出现 */
watch(() => props.open, (open) => {
    if (!open) {
        stopPolling()
        return
    }
    state.count = 3
    state.batchOpen = false
    state.codes = []
    state.copied = 0
    copiedIds.clear()
    loadHistory()
})

onBeforeUnmount(() => {
    stopPolling()
    window.clearTimeout(copiedTimer)
})
</script>

<style lang="scss" scoped>
.subtitle,
.step-desc {
  color: var(--gl-color-text-level2);
}

.prereq {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  background: var(--gl-color-bg-container1);
  color: var(--gl-color-text-level2);

  .prereq-icon {
    flex-shrink: 0;
    margin-top: 2px;
  }
}

.steps {
  list-style: none;
  margin: 20px 0 0;
  padding: 0;
}

.actions,
.more,
.codes-head,
.batch-panel,
.progress-head {
  display: flex;
  align-items: center;
  gap: 12px;
}

.codes-head {
  justify-content: space-between;
  color: var(--gl-color-text-level2);
}

.more {
  gap: 4px;
  font-size: 13px;
  color: var(--gl-color-text-level3);
}

.spacer {
  flex: 1;
}

.muted {
  color: var(--gl-color-text-level3);
}

.success {
  color: var(--gl-color-success-primary);
}

.link {
  padding: 4px 2px;
  border: none;
  background: none;
  font: inherit;
  font-size: 13px;
  color: var(--gl-color-brand-primary);
  cursor: pointer;

  &:hover {
    text-decoration: underline;
  }

  &:disabled {
    cursor: default;
    opacity: 0.5;
    text-decoration: none;
  }
}

.icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  fill: none;
  stroke: currentColor;
  stroke-width: 2.2;
  stroke-linecap: round;
  stroke-linejoin: round;

  &.brand {
    color: var(--gl-color-brand-primary);
  }
}

.spin {
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.code-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.code-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 8px 8px 12px;
  border: 1px solid var(--gl-color-line-border1);
  border-radius: 8px;
  background: var(--gl-color-bg-container1);

  &.settled {
    background: var(--gl-color-bg-surface1);
    border-color: var(--gl-color-line-divider1);
  }

  .code-no {
    width: 28px;
    flex-shrink: 0;
    font-size: 12px;
    color: var(--gl-color-text-level3);
  }

  .code-text {
    flex: 1;
    min-width: 0;
    padding: 0;
    background: none;
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 13px;
    line-height: 20px;
    color: var(--gl-color-text-level1);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .code-chip {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    height: 24px;
    padding: 0 8px;
    border-radius: 12px;
    font-size: 12px;
    background: var(--gl-color-bg-surface2);
    color: var(--gl-color-text-level3);

    &.joined {
      background: var(--gl-color-success-background);
      color: var(--gl-color-success-primary);
    }
  }

  .copy-btn {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }
}

.code-warn {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  line-height: 18px;
  color: var(--gl-color-warning-primary);
}

.batch-panel {
  padding: 10px 12px;
  border: 1px solid var(--gl-color-line-border1);
  border-radius: 8px;

  .count-input {
    width: 72px;
  }
}

.sketch {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  border: 1px dashed var(--gl-color-line-border1);
  border-radius: 8px;

  .sketch-head {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    color: var(--gl-color-text-level3);
  }

  .sketch-row {
    display: flex;
    gap: 8px;
  }

  .sketch-input,
  .sketch-btn {
    height: 32px;
    display: flex;
    align-items: center;
    border-radius: 6px;
    font-size: 12px;
  }

  .sketch-input {
    flex: 1;
    min-width: 0;
    padding: 0 10px;
    border: 1px solid var(--gl-color-line-border1);
    color: var(--gl-color-text-level3);
  }

  .sketch-btn {
    padding: 0 14px;
    background: var(--gl-color-brand-primary);
    color: var(--gl-color-text-white);
  }
}

.banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  border-radius: 8px;
  font-size: 13px;

  .icon {
    width: 16px;
    height: 16px;
  }
}

.banner-text {
  flex: 1;
}

.banner-wait {
  background: var(--gl-color-brand-background);
  color: var(--gl-color-brand-primary);
}

.banner-done {
  background: var(--gl-color-success-background);
  color: var(--gl-color-success-primary);
}

.banner-neutral {
  background: var(--gl-color-bg-container1);
  color: var(--gl-color-text-level2);
}

.progress-box {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  border-radius: 8px;
  background: var(--gl-color-bg-container1);
  font-size: 13px;
  color: var(--gl-color-text-level2);

  .progress-head {
    gap: 8px;
  }

  .progress-track {
    height: 6px;
    border-radius: 3px;
    background: var(--gl-color-line-divider1);
    overflow: hidden;
  }

  .progress-bar {
    height: 6px;
    border-radius: 3px;
    background: var(--gl-color-success-primary);
    transition: width 0.3s;
  }
}
</style>
