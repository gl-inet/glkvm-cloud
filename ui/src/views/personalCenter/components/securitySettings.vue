<!--
 * @Description: 安全设置卡片（含两步验证开关 + 信任设备列表）
-->
<template>
    <div class="pc-card">
        <div class="pc-card-header">
            <div class="pc-card-title">{{ $t('personalCenter.securitySettings') }}</div>
        </div>

        <div class="pc-section">
            <div class="pc-row">
                <div class="pc-row-text">
                    <div class="pc-row-title">{{ $t('personalCenter.twoFactorAuth') }}</div>
                    <div class="pc-row-desc">
                        {{ $t('personalCenter.twoFactorAuthDesc') }}
                        <template v-if="!isLocalUser">
                            ·
                            <span class="pc-row-warn">
                                {{ $t('personalCenter.twoFactorOnlyLocal', {
                                    provider: $t(AuthProviderLabelMap.get(props.profile?.authProvider) || 'login.local')
                                }) }}
                            </span>
                        </template>
                    </div>
                </div>
                <div class="pc-row-action">
                    <ASwitch
                        :checked="!!props.profile?.totpEnabled"
                        :disabled="!isLocalUser"
                        @change="onTwoFactorToggle"
                    />
                </div>
            </div>
        </div>

        <div v-if="props.profile?.totpEnabled" class="pc-section">
            <div class="pc-row">
                <div class="pc-row-text">
                    <div class="pc-row-title">{{ $t('personalCenter.trustedDevices') }}</div>
                    <div class="pc-row-desc">{{ $t('personalCenter.trustedDevicesDesc') }}</div>
                </div>
                <div class="pc-row-action">
                    <BaseButton size="middle" @click="loadTrustedDevices">{{ $t('common.refresh') }}</BaseButton>
                </div>
            </div>

            <div class="pc-trusted-list">
                <BaseTable
                    :data-source="state.trustedDevices"
                    :columns="trustedColumns"
                    :pagination="false"
                >
                    <template #deviceName="{ record }">
                        <div v-ellipsis>{{ record.deviceName || '--' }}</div>
                    </template>
                    <template #lastUsedAt="{ record }">{{ formatTs(record.lastUsedAt) }}</template>
                    <template #expiresAt="{ record }">{{ formatTs(record.expiresAt) }}</template>
                    <template #action="{ record }">
                        <a
                            rel="noopener noreferrer"
                            style="color: var(--gl-color-error-primary);"
                            @click="onRevoke(record)"
                        >{{ $t('personalCenter.revoke') }}</a>
                    </template>
                </BaseTable>
            </div>
        </div>

        <Enable2faDialog
            v-model:open="state.enableOpen"
            :username="props.profile?.username"
            @handleApply="onEnabled"
        />
        <Disable2faDialog
            v-model:open="state.disableOpen"
            @handleApply="onDisabled"
        />
    </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import dayjs from 'dayjs'
import { message } from 'ant-design-vue'
import { BaseButton } from 'gl-web-main/components'
import { baseCustomModal } from 'gl-web-main'
import BaseTable from '@/components/base/baseTable.vue'
import Enable2faDialog from './enable2faDialog.vue'
import Disable2faDialog from './disable2faDialog.vue'
import { reqListTrustedDevices, reqRevokeTrustedDevice } from '@/api/personal'
import { PersonalProfile, TrustedDevice } from '@/models/personal'
import { AuthProviderEnum, AuthProviderLabelMap } from '@/models/userManage'
import { t } from '@/hooks/useLanguage'

const props = defineProps<{ profile: PersonalProfile | null }>()
const emits = defineEmits<{ (e: 'refresh'): void }>()

const state = reactive({
    enableOpen: false,
    disableOpen: false,
    trustedDevices: [] as TrustedDevice[],
})

const isLocalUser = computed(() => props.profile?.authProvider === AuthProviderEnum.LOCAL)

const trustedColumns = computed(() => [
    { title: t('personalCenter.deviceName'), dataIndex: 'deviceName', ellipsis: true },
    { title: t('personalCenter.ipAddress'), dataIndex: 'ip', width: 160 },
    { title: t('personalCenter.lastUsed'), dataIndex: 'lastUsedAt', width: 180 },
    { title: t('personalCenter.expiresAt'), dataIndex: 'expiresAt', width: 180 },
    { title: t('common.action'), dataIndex: 'action', width: 120 },
] as any[])

const formatTs = (ts?: number) => {
    if (!ts || ts <= 0) return '--'
    return dayjs(ts * 1000).format('YYYY-MM-DD HH:mm:ss')
}

// Switch handler — open enable / disable dialog instead of toggling state directly,
// so the user must verify with a TOTP code in either direction.
const onTwoFactorToggle = (checked: boolean | string | number) => {
    if (!isLocalUser.value) return
    if (checked) {
        state.enableOpen = true
    } else {
        state.disableOpen = true
    }
}

const onEnabled = () => {
    message.success(t('common.success'))
    emits('refresh')
    loadTrustedDevices()
}

const onDisabled = () => {
    message.success(t('common.success'))
    state.trustedDevices = []
    emits('refresh')
}

const loadTrustedDevices = async () => {
    if (!props.profile?.totpEnabled) {
        state.trustedDevices = []
        return
    }
    try {
        const res = await reqListTrustedDevices()
        state.trustedDevices = res.data.items || []
    } catch {
        state.trustedDevices = []
    }
}

const onRevoke = (dev: TrustedDevice) => {
    baseCustomModal({
        type: 'confirm',
        title: t('personalCenter.revoke'),
        content: t('personalCenter.revokeConfirm'),
        onOk: async () => {
            await reqRevokeTrustedDevice(dev.id)
            message.success(t('common.success'))
            await loadTrustedDevices()
        },
    })
}

watch(() => props.profile?.totpEnabled, (v) => {
    if (v) loadTrustedDevices()
}, { immediate: true })
</script>

<style scoped lang="scss">
.pc-card {
  background-color: var(--gl-color-bg-surface1);
  border-radius: 10px;
  padding: 20px 24px;
}

.pc-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.pc-card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--gl-color-text-level1);
}

.pc-section {
  padding: 12px 0;
  border-top: 1px solid var(--gl-color-line-divider2);
}

.pc-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 24px;
}

.pc-row-text {
  flex: 1;
  min-width: 0;
}

.pc-row-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--gl-color-text-level1);
}

.pc-row-desc {
  margin-top: 4px;
  font-size: 12px;
  color: var(--gl-color-text-level3);
}

.pc-row-warn {
  color: var(--gl-color-warning-primary);
}

.pc-row-action {
  flex-shrink: 0;
}

.pc-trusted-list {
  margin-top: 16px;
}
</style>
