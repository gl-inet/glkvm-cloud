<!--
 * @Description: 添加设备弹窗里的一步：序号（完成后打勾）、连到下一步的竖线和标题，内容放默认插槽
-->
<template>
    <li class="step" :class="[`step-${props.state}`, { 'step-last': props.last }]">
        <div class="step-rail">
            <span class="step-dot">
                <BaseSvg v-if="props.state === 'done'" name="gl-icon-check-regular" :size="14"></BaseSvg>
                <template v-else>{{ props.index }}</template>
            </span>
            <span v-if="!props.last" class="step-line"></span>
        </div>
        <div class="step-body">
            <div class="step-title">{{ props.title }}</div>
            <slot></slot>
        </div>
    </li>
</template>

<script setup lang="ts">
const props = defineProps<{
    index: number
    /** todo 未到，active 进行中，done 已完成 */
    state: 'todo' | 'active' | 'done'
    title: string
    /** 最后一步不画竖线 */
    last?: boolean
}>()
</script>

<style lang="scss" scoped>
.step {
  display: flex;
  gap: 14px;
}

.step-rail {
  width: 24px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.step-dot {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
  background: var(--gl-color-bg-surface2);
  color: var(--gl-color-text-level3);
}

.step-line {
  width: 2px;
  flex: 1;
  min-height: 16px;
  margin: 6px 0;
  background: var(--gl-color-line-divider1);
}

.step-body {
  flex: 1;
  min-width: 0;
  padding-bottom: 24px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.step-title {
  font-size: 15px;
  font-weight: 600;
  line-height: 24px;
  color: var(--gl-color-text-level1);
}

.step-active .step-dot {
  background: var(--gl-color-brand-primary);
  color: var(--gl-color-text-white);
}

.step-done .step-dot {
  background: var(--gl-color-success-background);
  color: var(--gl-color-success-primary);
}

.step-todo .step-title {
  color: var(--gl-color-text-level3);
}

.step-last .step-body {
  padding-bottom: 20px;
}
</style>
