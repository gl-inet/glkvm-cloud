<!--
 * @Description: Device Event Logs page — split into two tabs
 *   - device:  device online/offline events
 *   - access:  remote SSH/Web session events
-->
<template>
    <div class="device-log-list">
        <div class="device-log-list-content">
            <BaseLoadingContainer :spinning="state.loading">
                <div class="device-log-list-view full-height">
                    <div class="header">
                        <div class="header-left">
                            <a-input-search
                                v-model:value="state.mac"
                                :placeholder="$t('deviceLog.macPlaceholder')"
                                style="width: 240px; margin-right: 8px;"
                                allowClear
                                @search="handleSearch"
                            />
                            <a-range-picker
                                v-model:value="state.dateRange"
                                :show-time="rangeShowTime"
                                format="YYYY-MM-DD HH:mm"
                                style="width: 340px; margin-right: 8px;"
                                :placeholder="[$t('deviceLog.createTime'), $t('deviceLog.endTime')]"
                                @change="handleSearch"
                            />
                            <BaseButton
                                style="margin-left: 8px;"
                                @click="handleSearch"
                            >
                                {{ $t('common.refresh') }}
                            </BaseButton>
                        </div>
                    </div>
                    <a-tabs
                        v-model:activeKey="state.activeTab"
                        class="log-tabs"
                        @change="handleTabChange"
                    >
                        <a-tab-pane key="device" :tab="$t('deviceLog.tabs.device')">
                            <div class="content">
                                <BaseTable
                                    :data-source="state.items"
                                    :columns="deviceColumns"
                                >
                                    <template #createdAt="{ record }">
                                        {{ formatTime(record.createdAt) }}
                                    </template>
                                    <template #eventType="{ record }">
                                        <BaseTag
                                            :style="{
                                                backgroundColor: eventTypeColor(record.eventType).bg,
                                                color: eventTypeColor(record.eventType).fg
                                            }"
                                        >
                                            {{ eventTypeLabel(record.eventType) }}
                                        </BaseTag>
                                    </template>
                                </BaseTable>
                            </div>
                        </a-tab-pane>
                        <a-tab-pane key="access" :tab="$t('deviceLog.tabs.access')">
                            <div class="content">
                                <BaseTable
                                    :data-source="state.items"
                                    :columns="accessColumns"
                                >
                                    <template #createdAt="{ record }">
                                        {{ formatTime(record.createdAt) }}
                                    </template>
                                    <template #endedAt="{ record }">
                                        <span v-if="record.endedAt > 0">{{ formatTime(record.endedAt) }}</span>
                                        <span v-else>-</span>
                                    </template>
                                    <template #duration="{ record }">
                                        <span v-if="record.endedAt > 0 && record.createdAt > 0">{{ formatDuration(record.endedAt - record.createdAt) }}</span>
                                        <span v-else>-</span>
                                    </template>
                                    <template #eventType="{ record }">
                                        <BaseTag
                                            :style="{
                                                backgroundColor: eventTypeColor(record.eventType).bg,
                                                color: eventTypeColor(record.eventType).fg
                                            }"
                                        >
                                            {{ eventTypeLabel(record.eventType) }}
                                        </BaseTag>
                                    </template>
                                    <template #actorName="{ record }">
                                        <span>{{ record.actorName || '-' }}</span>
                                    </template>
                                    <template #detail="{ record }">
                                        <template v-if="record.eventType !== 'remote_control'">
                                            <div v-ellipsis>{{ formatDetail(record.detail) }}</div>
                                        </template>
                                        <span v-else>-</span>
                                    </template>
                                </BaseTable>
                            </div>
                        </a-tab-pane>
                    </a-tabs>
                    <div class="pagination flex-end items-end">
                        <BasePagination
                            :total="state.total"
                            :pageSize="state.pageSize"
                            v-model:current="state.page"
                            @change="fetchLogs"
                        />
                    </div>
                </div>
            </BaseLoadingContainer>
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive } from 'vue'
import dayjs, { Dayjs } from 'dayjs'
import { TableColumnType } from 'ant-design-vue'
import BaseLoadingContainer from '@/components/base/baseLoadingContainer.vue'
import BasePagination from '@/components/base/basePagination.vue'
import BaseTable from '@/components/base/baseTable.vue'
import { BaseTag } from 'gl-web-main/components'
import { t } from '@/hooks/useLanguage'
import { reqListDeviceEventLogs, type DeviceEventLog } from '@/api/deviceLog'

type TabKey = 'device' | 'access'

const TAB_TYPES: Record<TabKey, string[]> = {
    device: ['device_online', 'device_offline'],
    access: ['remote_ssh', 'remote_web', 'remote_control'],
}

// Range picker time config:
//   - format: 'HH:mm'  hides the seconds column in the time panel
//   - defaultValue: pre-fills start time as 00:00 and end time as 23:59:59 so
//     the user can just pick two dates without touching the time column.
//     Built via startOf/endOf to avoid needing the customParseFormat plugin.
const rangeShowTime = {
    format: 'HH:mm',
    defaultValue: [
        dayjs().startOf('day'),
        dayjs().endOf('day'),
    ],
}

const state = reactive({
    loading: false,
    activeTab: 'device' as TabKey,
    mac: '',
    dateRange: null as [Dayjs, Dayjs] | null,
    page: 1,
    pageSize: 20,
    total: 0,
    items: [] as DeviceEventLog[],
})

const deviceColumns = computed<TableColumnType[]>(() => [
    { title: t('deviceLog.createTime'), dataIndex: 'createdAt' },
    { title: t('deviceLog.eventType'), dataIndex: 'eventType' },
    { title: t('deviceLog.mac'), dataIndex: 'deviceMac', ellipsis: true },
])

const accessColumns = computed<TableColumnType[]>(() => [
    { title: t('deviceLog.createTime'), dataIndex: 'createdAt', width: 170 },
    { title: t('deviceLog.endTime'), dataIndex: 'endedAt', width: 170 },
    { title: t('deviceLog.duration'), dataIndex: 'duration', width: 100 },
    { title: t('deviceLog.eventType'), dataIndex: 'eventType', width: 140 },
    { title: t('deviceLog.mac'), dataIndex: 'deviceMac', width: 140, ellipsis: true },
    { title: t('deviceLog.actor'), dataIndex: 'actorName', width: 120, ellipsis: true },
    { title: t('deviceLog.clientIp'), dataIndex: 'clientIp', width: 130 },
    { title: t('deviceLog.detail'), dataIndex: 'detail', width: 180, ellipsis: true },
])

const eventTypeLabel = (type: string): string => {
    switch (type) {
    case 'device_online':   return t('deviceLog.event.deviceOnline')
    case 'device_offline':  return t('deviceLog.event.deviceOffline')
    case 'remote_ssh':      return t('deviceLog.event.remoteSsh')
    case 'remote_web':      return t('deviceLog.event.remoteWeb')
    case 'remote_control':  return t('deviceLog.event.remoteControl')
    default:                return type
    }
}

const eventTypeColor = (type: string): { bg: string; fg: string } => {
    switch (type) {
    case 'device_online':
        return { bg: 'var(--gl-color-success-background)', fg: 'var(--gl-color-success-primary)' }
    case 'device_offline':
        return { bg: 'var(--gl-color-error-background)',   fg: 'var(--gl-color-error-primary)' }
    case 'remote_ssh':
        return { bg: 'var(--gl-color-primary-background)', fg: 'var(--gl-color-primary)' }
    case 'remote_web':
        return { bg: 'var(--gl-color-warning-background)', fg: 'var(--gl-color-warning-primary)' }
    case 'remote_control':
        return { bg: 'var(--gl-color-primary-background)', fg: 'var(--gl-color-primary)' }
    default:
        return { bg: 'var(--gl-color-bg-surface2)',        fg: 'var(--gl-color-text-primary)' }
    }
}

const formatTime = (ts: number) => {
    if (!ts) return '-'
    return dayjs.unix(ts).format('YYYY-MM-DD HH:mm:ss')
}

/** Format a duration in seconds into a human-readable string, e.g. "1h 23m 45s" */
const formatDuration = (seconds: number): string => {
    if (seconds < 0) return '-'
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const s = seconds % 60
    if (h > 0) return `${h}h ${m}m ${s}s`
    if (m > 0) return `${m}m ${s}s`
    return `${s}s`
}

/**
 * Parse the JSON detail blob into a human-readable string.
 * e.g. {"addr":"127.0.0.1:443","proto":"https"} → "https://127.0.0.1:443"
 */
const formatDetail = (raw: string): string => {
    if (!raw) return '-'
    try {
        const obj = JSON.parse(raw)
        if (obj.proto && obj.addr) {
            return `${obj.proto}://${obj.addr}`
        }
        // For other JSON shapes, show key=value pairs
        return Object.entries(obj).map(([k, v]) => `${k}=${v}`).join(', ')
    } catch {
        return raw
    }
}

const fetchLogs = async () => {
    state.loading = true
    try {
        const res = await reqListDeviceEventLogs({
            mac:      state.mac.trim(),
            types:    TAB_TYPES[state.activeTab],
            from:     state.dateRange?.[0] ? state.dateRange[0].unix() : undefined,
            to:       state.dateRange?.[1] ? state.dateRange[1].unix() : undefined,
            page:     state.page,
            pageSize: state.pageSize,
        })
        state.items = res.data.items || []
        state.total = res.data.total || 0
    } finally {
        state.loading = false
    }
}

const handleSearch = () => {
    state.page = 1
    fetchLogs()
}

const handleTabChange = () => {
    state.page = 1
    state.items = []
    state.total = 0
    fetchLogs()
}

onMounted(() => {
    fetchLogs()
})
</script>

<style scoped lang="scss">
.device-log-list {
  height: 100%;
  background-color: var(--gl-color-bg-surface1);
  border-radius: 10px;
  padding: 20px 24px;
  .device-log-list-content {
    height: 100%;

    .device-log-list-view {
      height: 100%;
      display: flex;
      flex-direction: column;
      .header {
        height: 36px;
        margin-bottom: 16px;
        display: flex;
        align-items: center;
      }
      .header-left {
        display: flex;
        align-items: center;
      }
      .log-tabs {
        flex: 1;
        min-height: 0;
        display: flex;
        flex-direction: column;
        :deep(.ant-tabs-content-holder) {
          flex: 1;
          min-height: 0;
        }
        :deep(.ant-tabs-content) {
          height: 100%;
        }
        :deep(.ant-tabs-tabpane) {
          height: 100%;
        }
      }
      .content {
        overflow-x: hidden;
        overflow-y: auto;
        height: 100%;
      }
      .pagination {
        height: 40px;
      }
      .muted {
        color: var(--gl-color-text-secondary);
      }
    }
  }
}
</style>
