<!--
 * @Description: 添加设备弹窗的脚本页：给固件低于 V1.11.0、不能用接入码的 GL KVM 用
-->
<template>
    <div class="legacy-script">
        <AAlert class="cloud-warn" type="warning" showIcon :message="$t('device.scriptCloudExclusive')" />

        <ol class="script-steps">
            <li><BaseText type="body-r">{{ $t('device.scriptStepCloudOff') }}</BaseText></li>
            <li><BaseText type="body-r">{{ $t('device.scriptStepRun') }}</BaseText></li>
            <li><BaseText type="body-r">{{ $t('device.scriptStepOnline') }}</BaseText></li>
        </ol>

        <AAlert v-if="state.error" type="error" showIcon :message="state.error" />
        <template v-else>
            <div class="script-head">
                <BaseText type="body-m">{{ $t('device.scriptTitle') }}</BaseText>
                <BaseButton size="small" :disabled="!state.script" @click="copyScript">
                    {{ state.copied ? $t('device.copied') : $t('device.copy') }}
                </BaseButton>
            </div>
            <pre class="script-box">{{ state.loading ? '…' : state.script }}</pre>
            <BaseText el="div" type="caption-r" class="muted">{{ $t('device.scriptTokenWarning') }}</BaseText>
            <BaseText v-if="state.fingerprint" el="div" type="caption-r" class="muted">
                {{ $t('device.scriptCAFingerprint') }}
                <code class="fingerprint">{{ state.fingerprint }}</code>
            </BaseText>
        </template>
    </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive } from 'vue'
import { copyText } from 'gl-web-main'
import { message } from 'ant-design-vue'
import { getLegacyScriptApi } from '@/api/device'
import { t } from '@/hooks/useLanguage'

const state = reactive({
    loading: true,
    script: '',
    fingerprint: '',
    error: '',
    copied: false,
})

let copiedTimer: number | undefined

const load = async () => {
    state.loading = true
    try {
        const res = await getLegacyScriptApi()
        state.script = res.data.script
        state.fingerprint = res.data.caFingerprint
    } catch (error) {
        state.error = t('device.scriptLoadFailed')
        console.error('Failed to load the add device script:', error)
    } finally {
        state.loading = false
    }
}

const copyScript = async () => {
    try {
        await copyText(state.script)
    } catch {
        message.error(t('common.copyFailed'))
        return
    }
    state.copied = true
    window.clearTimeout(copiedTimer)
    copiedTimer = window.setTimeout(() => {
        state.copied = false
    }, 1600)
}

onMounted(load)
onBeforeUnmount(() => window.clearTimeout(copiedTimer))
</script>

<style lang="scss" scoped>
.legacy-script {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 16px;
}

.script-steps {
  margin: 0;
  padding-left: 20px;
  color: var(--gl-color-text-level2);

  li + li {
    margin-top: 4px;
  }
}

.script-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.script-box {
  max-height: 240px;
  margin: 0;
  padding: 10px 12px;
  overflow: auto;
  border-radius: 8px;
  background: var(--gl-color-bg-container1);
  color: var(--gl-color-text-level1);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre;
  tab-size: 4;
}

.muted {
  color: var(--gl-color-text-level3);
}

.fingerprint {
  word-break: break-all;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
</style>
