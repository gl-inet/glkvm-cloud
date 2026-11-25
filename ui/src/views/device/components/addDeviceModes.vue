<!--
 * @Description: 添加设备的两种方式：接入码（默认），以及平台还接受旧 token 接入时，给固件低于 V1.11.0 的设备用的脚本
-->
<template>
    <ARadioGroup v-if="props.script" v-model:value="mode" buttonStyle="solid" class="mode-switch">
        <ARadioButton value="code">{{ $t('device.addByAccessCode') }}</ARadioButton>
        <ARadioButton value="script">{{ $t('device.addByScript', { version: props.minFirmware }) }}</ARadioButton>
    </ARadioGroup>
    <LegacyScriptPanel v-if="props.script && mode === 'script'" />
    <slot v-else name="code"></slot>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import LegacyScriptPanel from './legacyScriptPanel.vue'

const props = defineProps<{
    /** 平台提供脚本 */
    script: boolean
    /** 支持接入码的最低固件版本 */
    minFirmware: string
}>()

/** 弹窗关闭时内容会销毁，下次打开回到接入码 */
const mode = ref<'code' | 'script'>('code')
</script>

<style lang="scss" scoped>
.mode-switch {
  margin-bottom: 16px;
}
</style>
