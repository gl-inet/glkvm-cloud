<!--
 * @Author: shufei.han
 * @Date: 2025-06-09 09:22:43
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 11:01:41
 * @FilePath: \glkvm-cloud\ui\src\views\device\devicePage.vue
 * @Description: 设备模块入口页面
-->
<template>
    <BasePage v-if="!state.hasFetchedDevice">
        <div  class="full-height flex">
            <BaseLoading antdMode block>
            </BaseLoading>
        </div>
    </BasePage>
    <BasePage v-else-if="!deviceStore.state.hasDevice">
        <NoDevicePage />
    </BasePage>

    <div class="device-list-container" v-else>
        <div class="out-device-list-header">
            <div class="left">
                <BaseText type="large-title-m">{{ $t('device.devices') + '(' + deviceStore.deviceList.length + ')' }}</BaseText>
            </div>
        </div>
        <div class="device-list">
            <div class="device-list-content">
                <DeviceListView />
            </div> 
        </div>
    </div>
</template> 

<script setup lang="ts">
import BasePage from '@/components/base/basePage.vue'
import NoDevicePage from './components/noDevicePage.vue'
import { useDeviceStore } from '@/stores/modules/device'
import DeviceListView from './components/deviceListView.vue'
import { reactive, onBeforeUnmount, onMounted } from 'vue'
import { BaseLoading } from 'gl-web-main/components'

const deviceStore = useDeviceStore()

const state = reactive({
    remoteLoadingDeviceMacs: new Set<string>(),
    /** 是否已经第一次获取完毕 */
    hasFetchedDevice: false,
})

deviceStore.startPolling()

onBeforeUnmount(() => {
    deviceStore.stopPolling()
    deviceStore.state.searchText = ''
    deviceStore.state.deviceGroupId = undefined
})

onMounted(async () => { 
    await deviceStore.getDeviceList(false, true)
    state.hasFetchedDevice = true
})
</script> 

<style lang="scss" scoped>
.device-list-container {
    height: 100%;
    padding: 20px 24px;
    background-color: var(--gl-color-bg-page);

    .out-device-list-header {
        height: 48px;
        margin-bottom: 16px;
        padding: 0 12px;
        display: flex;
        justify-content: space-between;
        align-items: center;

        .left {
            display: flex;
            align-items: center;
        }
    }

    .device-list {
        height: calc(100% - 64px);
        background-color: var(--gl-color-bg-surface1);
        border-radius: 10px;
        padding: 20px 24px;
        .device-list-content {
            height: 100%;
        }
    }
}
</style>