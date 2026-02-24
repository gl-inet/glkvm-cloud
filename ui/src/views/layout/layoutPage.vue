<!--
 * @Author: LPY
 * @Date: 2025-05-30 15:20:13
 * @LastEditors: LPY
 * @LastEditTime: 2026-01-29 10:40:50
 * @FilePath: \glkvm-cloud\ui\src\views\layout\layoutPage.vue
 * @Description: 路由总集成页
-->
<template>
    <div class="layout">
        <LayHeader />
        <div class="lay-body">
            <LaySide class="lay-side" />
            <LayContent class="lay-content" />
        </div>
    </div>
</template>

<script setup lang="ts">
import LayContent from './layContent/layContent.vue'
import LayHeader from './layHeader/layHeader.vue'
import LaySide from './laySide/laySide.vue'
</script>

<style scoped lang="scss">
.layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  min-width: 800px;
  overflow: hidden;

  .lay-body {
    display: flex;
    flex: 1;
    width: 100%;
    overflow: hidden;

    .lay-side {
      height: 100%;
      flex-shrink: 0;
    }

    .lay-content {
      flex: 1;
      height: 100%;
      overflow: auto;
    }
  }
}
</style>