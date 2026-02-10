<!--
 * @Author: shufei.han
 * @Date: 2025-06-12 15:56:06
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-05 16:19:52
 * @FilePath: \glkvm-cloud\ui\src\views\device\components\noDevicePage.vue
 * @Description: 
-->
<template>
    <div class="flex full-height">
        <div class="flex flex-column">
            <img :src="noDeviceSvg" width="240" height="240" alt="no device yet">
            <BaseText class="text-center no-device" type='large-title-m'>{{ $t('device.noDevice') }}</BaseText>
            <BaseText class="text-center no-device-tip" type='head-r' variant="level2">{{ $t('device.noDeviceTip') }}</BaseText>
            <BaseButton v-if="hasPermission(PermissionEnum.DEVICE_WRITE)" medium primary class="text-center" @click="addDevice">
                <GlSvg name="gl-icon-plus-regular" style="color: var(--gl-color-text-white);margin-right: 4px;"></GlSvg>
                <span style="padding-left: 4px;">{{ $t('device.addDevice') }}</span>
            </BaseButton>
        </div>

        <!-- 添加设备弹窗 -->
        <AddDeviceDialog
            v-model:open="state.addDeviceOpen" 
        />
    </div>
</template> 

<script setup lang="ts">
import noDeviceSvg from '@/assets/svg/no-device.svg'
import AddDeviceDialog from './addDeviceDialog.vue'
import { reactive } from 'vue'
import { hasPermission } from '@/utils/permission'
import { PermissionEnum } from '@/models/permission'
import { GlSvg } from 'gl-web-main/components'

const state = reactive({
    addDeviceOpen: false,
})

const addDevice = () => { 
    state.addDeviceOpen = true
}
</script> 

<style lang="scss" scoped>
.no-device {
    margin: 24px 0 4px;
}
.no-device-tip {
    margin-bottom: 32px;
}
</style>