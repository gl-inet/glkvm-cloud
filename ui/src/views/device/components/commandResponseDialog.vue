<!--
 * @Author: LPY
 * @Date: 2025-08-26 11:15:08
 * @LastEditors: LPY
 * @LastEditTime: 2025-08-26 16:51:00
 * @FilePath: \glkvm-cloud\web-ui\src\views\device\components\commandResponseDialog.vue
 * @Description: 
-->
<template>
    <BaseModal
        :width="600"
        :open="props.open"
        :title="$t('device.commandResponse')"
        destroyOnClose
        @close="emits('update:open', false)"
        :showFooter="false"
    >
        <div v-if="!state.isAllRes" class="loading">
            <a-progress 
                :percent="progress"
                :size="10"
                trailColor="var(--gl-color-bg-container1)"
                strokeColor="var(--gl-color-brand-primary)"
            />
            <BaseText style="margin-top: 12px;" center>{{ $t('device.commandResponseLoadingTips') }}</BaseText>
        </div>
        <div v-else class="table">
            <a-tabs v-model:activeKey="state.activeKey">
                <a-tab-pane key="success" :tab="$t('common.success') + ' ' + successResList.length">
                    <BaseTable
                        :data-source="successResList"
                        :columns="successResColumns"
                        rowKey="id"
                    >
                        <template #index="{ record }">
                            <div class="flex-start">
                                {{ successResList.findIndex(item => item.id === record.id) + 1 }}
                            </div>
                        </template>
                        <template #action="{ record }">
                            <div class="flex-start">
                                <a target="_blank" rel="noopener noreferrer" @click="handleOpenDetail(record)">{{ $t('common.detail') }}</a>
                            </div>
                        </template>
                    </BaseTable>
                </a-tab-pane>
                <a-tab-pane key="failed" :tab="$t('common.failed') + ' ' + failedResList.length">
                    <BaseTable
                        :data-source="failedResList"
                        :columns="failedResColumns"
                        rowKey="id"
                    >
                        <template #index="{ record }">
                            <div class="flex-start">
                                {{ failedResList.findIndex(item => item.id === record.id) + 1 }}
                            </div>
                        </template>
                        <template #action="{ record }">
                            <div class="flex-start">
                                <a target="_blank" rel="noopener noreferrer" @click="handleOpenDetail(record)">{{ $t('common.detail') }}</a>
                            </div>
                        </template>
                    </BaseTable>
                </a-tab-pane>
            </a-tabs>
        </div>

        <!-- 详情弹窗 -->
        <CmdResDetailDialog
            v-model:open="state.cmdResOpen"
            :type="state.activeKey"
            :res="state.currentRes"
        />
    </BaseModal>
</template>

<script setup lang="ts">
import { reqExecuteCommand } from '@/api/device'
import BaseTable from '@/components/base/baseTable.vue'
import { t } from '@/hooks/useLanguage'
import { DeviceInfo, ExecuteCommandFormData } from '@/models/device'
import { useFakeUpgradeProgress } from 'gl-web-main'
import BaseModal from '@/components/base/baseModalI18n.vue'
import { TableColumnType } from 'ant-design-vue'
import { computed, reactive, watch } from 'vue'
import CmdResDetailDialog from './cmdResDetailDialog.vue'

const props = defineProps<{ open: boolean, selection: DeviceInfo[], formData: ExecuteCommandFormData | undefined }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
}>()

const { progress, completeProgress } = useFakeUpgradeProgress(5)

const state = reactive({
    isAllRes: false, // 是否请求完成

    total: 0,
    fail: 0,
    responses: [],

    activeKey: 'success',
    currentRes: undefined,
    cmdResOpen: false,
})

watch(() => state.responses, (val) => {
    if (val.length && (val.length === state.total)) {
        completeProgress()
        setTimeout(() => {
            state.isAllRes = true
        }, 1000)
    }
}, {deep: true, immediate: true})

// 成功列表
const successResList = computed(() => {
    return state.responses.filter(response => !response.err)
})
const successResColumns = computed<TableColumnType[]>(() => { 
    return [
        {title: '#', dataIndex: 'index', ellipsis: true},
        {title: t('device.deviceID'), dataIndex: 'id', ellipsis: true},
        {title: t('device.code'), dataIndex: 'code', ellipsis: true},
        {title: t('common.action'), dataIndex: 'action'},
    ]
})

// 失败列表
const failedResList = computed(() => {
    return state.responses.filter(response => response.err && response.err !== 0)
})
const failedResColumns = computed<TableColumnType[]>(() => { 
    return [
        {title: '#', dataIndex: 'index', ellipsis: true, width: 50},
        {title: t('device.deviceID'), dataIndex: 'id', ellipsis: true, width: 100},
        {title: t('device.errorCode'), dataIndex: 'err', ellipsis: true, width: 80},
        {title: t('device.errorMessage'), dataIndex: 'msg', ellipsis: true},
        {title: t('common.action'), dataIndex: 'action', width: 80},
    ]
})

/** 打开响应详情 */
const handleOpenDetail = (record: any) => {
    state.currentRes = record
    state.cmdResOpen = true
}

const init = () => {
    // 调用接口进行下发配置
    const selections = props.selection.filter(item => item.proto > 4)
    state.total = selections.length
    selections.map(device => {
        const data = {
            id: device.id,
            group: '',
            wait: props.formData.wait,
            cmd: props.formData.cmd,
            params: props.formData.params,
            username: props.formData.username,
        }

        reqExecuteCommand(data).then(res => {
            if (props.formData.wait === 0) {
                state.responses.push({
                    err: 0,
                    msg: '',
                    id: device.id,
                    code: 0,
                    stdout: '',
                    stderr: '',
                })
            } else {
                const resp = res.info

                if (resp.err && resp.err !== 0) {
                    state.fail++
                    resp.stdout = ''
                    resp.stderr = ''
                } else {
                    resp.stdout = window.atob(resp.stdout || '')
                    resp.stderr = window.atob(resp.stderr || '')
                }

                resp.id = device.id
                state.responses.push(resp)
            }
        })
    })
}

init()
</script>

<style scoped lang="scss">
.loading {
    width: 552px;
    margin: 40px 0;

    .ant-progress-line {
        margin: 0;
    }
}
</style>