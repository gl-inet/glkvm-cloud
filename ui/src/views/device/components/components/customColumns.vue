<!--
 * @Author: LPY
 * @Date: 2026-03-09 14:27:32
 * @LastEditors: LPY
 * @LastEditTime: 2026-03-10 11:48:15
 * @FilePath: \glkvm-cloud\ui\src\views\device\components\components\customColumns.vue
 * @Description: 自定义table列组件
-->
<template>
    <APopover trigger="click" placement="bottom" :arrow="false" overlayClassName="custom-columns-popper" @openChange="handleOpenChange">
        <template #content>
            <div class="top-tips">
                <BaseText type="head-m">{{ $t('device.customColumns') }}</BaseText>
                <GlSvg name="gl-icon-help" tooltip :size="20">{{ $t('device.dragColumnTips') }}</GlSvg>
            </div>
            <div class="dividing-line"></div>
            <div ref="customColumnsBoxRef" class="custom-columns-box">
                <div v-for="item in clonedColumns" :key="item.key" class="custom-columns-item" :style="{'cursor': dragging ? 'grabbing' : undefined}">
                    <div class="custom-columns-item-left">
                        <ACheckbox :checked="item.show" @change="handleColumnVisibilityChange(item)"></ACheckbox>
                        <BaseText v-ellipsis class="title">{{ item.title }}</BaseText>
                    </div>
                    <div class="custom-columns-item-right">
                        <GlSvg name="gl-icon-grip-dots-vertical-regular"></GlSvg>
                    </div>
                </div>
            </div>
        </template>
        <GlSvg name="gl-icon-gear-regular" :size="20" class="custom-columns-icon"></GlSvg>
    </APopover>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { GlSvg } from 'gl-web-main/components'
import Sortable from 'sortablejs'
import { LocalStorageKeys, useLocalStorage } from '@/hooks/useLocalStorage'
import { TableColumnType } from 'ant-design-vue'

// Avoid deep type instantiation by defining only the properties you use
interface CustomTableColumnType extends TableColumnType {
  show?: boolean
  title?: string
}

const props = withDefaults(defineProps<{
  storageName: LocalStorageKeys
  columns: CustomTableColumnType[]
  completeColumns: CustomTableColumnType[]
}>(), {
})

const emits = defineEmits<{
  (e: 'change', value: TableColumnType[]): void
}>()

const customColumnsBoxRef = ref<HTMLDivElement>()

const dragging = ref(false)

const clonedColumns = ref<CustomTableColumnType[]>([])

const initDragFn = () => {
    // 注册拖拽元素
    Sortable.create(customColumnsBoxRef.value, {
        group: 'columns',
        animation: 150,
        draggable: '.custom-columns-item',
        dragClass: 'custom-columns-item-dragging',
        forceFallback: true,
        handle: '.custom-columns-item-right',
        onStart () {
            dragging.value = true
        },
        onEnd: (evt) => handleSortEnd(evt),
    })
}

const handleSortEnd = ({
    oldIndex,
    newIndex,
    from,
    to,
}) => {
    dragging.value = false
    console.log(oldIndex, newIndex, from, to)
    const movedColumn = clonedColumns.value[oldIndex]
    clonedColumns.value.splice(oldIndex, 1)
    clonedColumns.value.splice(newIndex, 0, movedColumn)
    emits('change', clonedColumns.value)
    useLocalStorage(props.storageName).setValue(clonedColumns.value)
}

const handleColumnVisibilityChange = (column: CustomTableColumnType) => {
    column.show = !column.show
    // 找到对应的列并更新show属性
    emits('change', clonedColumns.value)
    useLocalStorage(props.storageName).setValue(clonedColumns.value)
}

const handleOpenChange = (open: boolean) => {
    if (open) {
        // 使用 nextTick 确保 DOM 已渲染
        nextTick(() => {
            if (customColumnsBoxRef.value) {
                initDragFn()

                // 初始化数据，若有存储，则以存储为准，否则使用 completeColumns 的默认值
                const storedColumns = useLocalStorage(props.storageName).getValue() as CustomTableColumnType[] | null
                if (storedColumns) {
                    clonedColumns.value = storedColumns
                } else {
                    clonedColumns.value = props.completeColumns.map(col => ({
                        ...col,
                        show: props.columns.find(c => c.key === col.key)?.show ?? true,
                    }))
                }
            }
        })
    }
}
</script>

<style scoped lang="scss">
.top-tips {
    width: 200px;
    padding: 0 8px;
    display: flex;
    justify-content: space-between;
}

.dividing-line {
    height: 1px;
    background-color: var(--gl-color-line-divider1);
    margin: 12px 0;
}

.custom-columns-box {
  width: 200px;
  .custom-columns-item {
    display: flex;
    justify-content: space-between;
    align-items: center;

    border-radius: 4px;
    height: 36px;
    padding: 0 8px;

    cursor: pointer;

    &:hover {
      background-color: var(--gl-color-bg-item-hover);
    }
    .custom-columns-item-left {
      display: flex;
      align-items: center;

      .title {
        max-width: 134px;
        padding-left: 10px;
      }
    }

    .custom-columns-item-right {
      display: flex;
      justify-content: center;
      align-items: center;
      width: 32px;
      height: 32px;
      cursor: grabbing;
    }
  }
}

.custom-columns-icon {
    display: inline-block;
    height: 20px;
    line-height: 20px;
    margin-right: 12px;
    cursor: pointer;
}

.custom-columns-item-dragging {
    box-shadow: 0px 3px 14px 0px rgba(0,0,0,0.25);
    opacity: 1 !important;
    background-color: var(--gl-color-bg-surface1);
}
</style>