<!--
 * @Description: 添加设备弹窗底部的接入码记录：默认折叠、前端分页，未使用的可以作废
-->
<template>
    <div class="history">
        <button type="button" class="history-toggle" :aria-expanded="state.open" @click="state.open = !state.open">
            <BaseText type="body-m">{{ $t('device.recentAccessCodes') }}</BaseText>
            <BaseText type="caption-r" class="muted">{{ unusedText }}</BaseText>
            <span class="spacer"></span>
            <BaseSvg name="gl-icon-chevron-down-regular" :size="16" class="chevron" :class="{ open: state.open }"></BaseSvg>
        </button>
        <div v-if="state.open" class="history-body">
            <BaseText v-if="!props.items.length" el="div" type="caption-r" class="muted">{{ $t('device.noAccessCodes') }}</BaseText>
            <template v-else>
                <div class="history-row history-head">
                    <span>{{ $t('device.codeCreatedAt') }}</span>
                    <span>{{ $t('device.codeStatus') }}</span>
                    <span>{{ $t('device.codeDevice') }}</span>
                    <span></span>
                </div>
                <div v-for="item in pagedItems" :key="item.id" class="history-row">
                    <span class="history-time">{{ formatTime(item.createdAt) }}</span>
                    <span>
                        <BaseTag :primary="item.status === 'unused'" :success="item.status === 'used'">
                            {{ $t(`device.accessCodeStatus.${item.status}`) }}
                        </BaseTag>
                    </span>
                    <span class="history-device">{{ item.devid || '—' }}</span>
                    <span class="history-op">
                        <APopconfirm
                            v-if="item.status === 'unused'"
                            :title="$t('device.revokeAccessCodeConfirm')"
                            @confirm="emits('revoke', item.id)"
                        >
                            <button type="button" class="link danger">{{ $t('device.revokeAccessCode') }}</button>
                        </APopconfirm>
                    </span>
                </div>
                <div v-if="props.items.length > PAGE_SIZE" class="history-pager">
                    <APagination
                        v-model:current="state.page"
                        size="small"
                        :pageSize="PAGE_SIZE"
                        :showSizeChanger="false"
                        :total="props.items.length"
                    />
                </div>
            </template>
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import dayjs from 'dayjs'
import { t } from '@/hooks/useLanguage'
import type { RegistrationCode } from '@/models/device'

/** 每页条数 */
const PAGE_SIZE = 5

const props = defineProps<{ items: RegistrationCode[] }>()

const emits = defineEmits<{
    (e: 'revoke', id: number): void;
}>()

const state = reactive({
    open: false,
    page: 1,
})

const pagedItems = computed(() => {
    const start = (state.page - 1) * PAGE_SIZE
    return props.items.slice(start, start + PAGE_SIZE)
})

const unusedText = computed(() => {
    const n = props.items.filter(item => item.status === 'unused').length
    return n ? t('device.unusedCodes', { n }) : t('device.noUnusedCodes')
})

const formatTime = (ts: number) => dayjs.unix(ts).format('YYYY-MM-DD HH:mm')

/** 新生成了接入码（最新的排在最前）就回到第一页 */
watch(() => props.items[0]?.id, () => {
    state.page = 1
})
</script>

<style lang="scss" scoped>
.history {
  border-top: 1px solid var(--gl-color-line-divider1);
}

.history-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 0;
  border: none;
  background: none;
  font: inherit;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.spacer {
  flex: 1;
}

.muted {
  color: var(--gl-color-text-level3);
}

.chevron {
  color: var(--gl-color-text-level3);
  transition: transform 0.15s;

  &.open {
    transform: rotate(180deg);
  }
}

.history-body {
  padding-bottom: 4px;
}

.history-row {
  display: grid;
  grid-template-columns: 136px 96px minmax(0, 1fr) 56px;
  gap: 12px;
  align-items: center;
  min-height: 40px;
  border-top: 1px solid var(--gl-color-line-divider1);
  font-size: 13px;
}

.history-head {
  min-height: 32px;
  border-top: none;
  font-size: 12px;
  color: var(--gl-color-text-level3);
}

.history-time {
  color: var(--gl-color-text-level2);
  font-variant-numeric: tabular-nums;
}

.history-device {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-op {
  justify-self: end;
}

.link {
  padding: 4px 2px;
  border: none;
  background: none;
  font: inherit;
  font-size: 13px;
  cursor: pointer;

  &.danger {
    color: var(--gl-color-error-primary);
  }

  &:hover {
    text-decoration: underline;
  }
}

.history-pager {
  display: flex;
  justify-content: flex-end;
  padding-top: 10px;
}
</style>
