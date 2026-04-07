<!--
 * @Description: 个人信息卡片
-->
<template>
    <div class="pc-card">
        <div class="pc-card-header">
            <div class="pc-card-title">{{ $t('personalCenter.personalInformation') }}</div>
            <BaseButton
                v-if="canEdit"
                size="middle"
                @click="state.editOpen = true">
                {{ $t('common.edit') }}
            </BaseButton>
        </div>

        <div class="pc-card-body">
            <div class="pc-info-item">
                <div class="pc-info-label">{{ $t('personalCenter.username') }}</div>
                <div class="pc-info-value">{{ props.profile?.username || '--' }}</div>
            </div>

            <div class="pc-info-item">
                <div class="pc-info-label">{{ $t('personalCenter.displayName') }}</div>
                <div class="pc-info-value">
                    <template v-if="props.profile?.displayName">{{ props.profile.displayName }}</template>
                    <span v-else class="pc-info-empty">{{ $t('personalCenter.notFilled') }}</span>
                </div>
            </div>

            <div class="pc-info-item">
                <div class="pc-info-label">{{ $t('personalCenter.authProvider') }}</div>
                <div class="pc-info-value">
                    {{ props.profile ? $t(AuthProviderLabelMap.get(props.profile.authProvider) || 'login.local') : '--' }}
                </div>
            </div>

            <div class="pc-info-item">
                <div class="pc-info-label">{{ $t('personalCenter.registrationTime') }}</div>
                <div class="pc-info-value">{{ formatTs(props.profile?.registrationTime) }}</div>
            </div>

            <div class="pc-info-item">
                <div class="pc-info-label">{{ $t('personalCenter.latestLoginTime') }}</div>
                <div class="pc-info-value">{{ formatTs(props.profile?.lastLoginTime) }}</div>
            </div>
        </div>

        <EditProfileDialog
            v-model:open="state.editOpen"
            :profile="props.profile"
            @handleApply="onApplied"
        />
    </div>
</template>

<script setup lang="ts">
import { computed, reactive } from 'vue'
import dayjs from 'dayjs'
import { BaseButton } from 'gl-web-main/components'
import EditProfileDialog from './editProfileDialog.vue'
import { PersonalProfile } from '@/models/personal'
import { AuthProviderEnum, AuthProviderLabelMap } from '@/models/userManage'

const props = defineProps<{ profile: PersonalProfile | null }>()

const emits = defineEmits<{ (e: 'refresh'): void }>()

const state = reactive({
    editOpen: false,
})

// 外部用户（OIDC/LDAP）的 displayName 由 IdP 同步，本地不能改
const canEdit = computed(() => {
    if (!props.profile) return false
    return props.profile.authProvider === AuthProviderEnum.LOCAL
})

const formatTs = (ts?: number | null) => {
    if (!ts || ts <= 0) return '--'
    return dayjs(ts * 1000).format('YYYY-MM-DD HH:mm:ss')
}

const onApplied = () => {
    emits('refresh')
}
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
  margin-bottom: 20px;
}

.pc-card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--gl-color-text-level1);
}

.pc-card-body {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  row-gap: 24px;
  column-gap: 24px;
}

.pc-info-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.pc-info-label {
  font-size: 13px;
  color: var(--gl-color-text-level3);
}

.pc-info-value {
  font-size: 14px;
  color: var(--gl-color-text-level1);
  word-break: break-all;
}

.pc-info-empty {
  color: var(--gl-color-text-level3);
}
</style>
