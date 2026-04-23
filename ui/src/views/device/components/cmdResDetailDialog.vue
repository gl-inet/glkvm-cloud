<!--
 * @Author: LPY
 * @Date: 2025-08-26 16:45:08
 * @LastEditors: LPY
 * @LastEditTime: 2025-08-26 17:05:54
 * @FilePath: \glkvm-cloud\web-ui\src\views\device\components\cmdResDetailDialog.vue
 * @Description: 执行命令结果详情弹窗
-->
<template>
    <BaseModal
        :width="672"
        :open="props.open"
        :title="$t('device.commandResponseDetail')"
        destroyOnClose
        @close="emits('update:open', false)"
        :showFooter="false"
    >
        <div class="id">
            <BaseText variant="level2" style="margin-right: 12px;">{{ $t('device.deviceID') }}</BaseText>
            <BaseText>{{ props.res.id }}</BaseText>
        </div>

        <BaseText style="margin: 20px 0 6px;">{{ props.type === 'success' ? $t('device.standardOutput') : $t('device.standardErrorOutput') }}</BaseText>
        <ATextarea :value="props.type === 'success' ? props.res.stdout : props.res.stderr" readonly :autoSize="{minRows:1, maxRows: 20}"></ATextarea>
    </BaseModal>
</template>

<script setup lang="ts">
import BaseModal from '@/components/base/baseModalI18n.vue'

const props = defineProps<{ open: boolean, type: string, res: any }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
}>()
</script>

<style scoped lang="scss">
.id {
  display: flex;
  align-items: center;
}
</style>