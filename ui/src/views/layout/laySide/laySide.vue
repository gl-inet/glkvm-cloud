<!--
 * @Author: LPY
 * @Date: 2025-05-30 15:20:53
 * @LastEditors: LPY
 * @LastEditTime: 2025-07-18 12:11:09
 * @FilePath: /kvm-cloud-frontend/src/views/layout/laySide/laySide.vue
 * @Description: 侧边集成页
-->
<template>
    <div class="scrollbar-container">
        <!-- 菜单主体 -->
        <div ref="scrollbarWrapperRef" :class="{'scrollbar-wrapper':true, 'scrollbar-wrapper-collapsed': appStore.isCollapse}">
            <!-- 展开状态按钮 -->
            <div v-if="!appStore.isCollapse" class="menu-list">
                <ul>
                    <li v-for="(routeItem, routeIndex) in state.menu" :key="routeIndex">
                        <div class="route-li" :class="{ active: isActive(routeItem) }" @click="handleSelectRoute(routeItem, routeIndex)">
                            <div class="route-info">
                                <div class="route-icon">
                                    <BaseSvg :name="routeItem.meta.icon" :size="18" />
                                </div>
                                <div class="route-title">
                                    {{ $t(routeItem.meta.title) }}
                                </div>
                            </div>
                            <div
                                v-if="routeItem.children && routeItem.children.length > 0"
                                class="chevron-down-icon"
                                :class="{'icon-active': (routeItem.redirect || routeItem.path) && routeItem.meta.whetherToExpandChildElements }"
                            >
                                <BaseSvg name="gl-icon-chevron-down-regular" />
                            </div>
                        </div>

                        <transition name="route-li">
                            <ul
                                class="child-route-list"
                                :style="{ 'max-height': routeItem.meta.whetherToExpandChildElements ? calculateHeight(routeItem.children.length) : '0px' }"
                            >
                                <li v-for="(child, childIndex) in routeItem.children" :key="childIndex">
                                    <div
                                        class="route-li"
                                        style="padding: 0px 10px 0px 30px"
                                        :class="{ active: isActive(child) }"
                                        @click="handleSelectChildRoute(child, routeIndex)"
                                    >
                                        <div class="route-info">
                                            <div class="route-icon">
                                                <BaseSvg :name="child.meta.icon" :size="18"/>
                                            </div>
                                            <div class="route-title">
                                                {{ $t(child.meta.title) }}
                                            </div>
                                        </div>
                                    </div>
                                </li>
                            </ul>
                        </transition>
                    </li>
                </ul>
            </div>

            <!-- 收缩状态按钮 -->
            <ul v-else class="little-icon-list">
                <li v-for="(routeItem, routeIndex) in state.menu" :key="routeIndex">
                    <LaySidePopover ref="laySidePopoverRefs">
                        <template v-slot:content>
                            <div v-if="routeItem.children && routeItem.children.length > 0" style="z-index: 1000;">
                                <div class="little-title">
                                    {{ $t(routeItem.meta.title) }}
                                </div>
                                <ul>
                                    <li v-for="(child, childIndex) in routeItem.children" :key="childIndex">
                                        <div
                                            class="route-li"
                                            style="padding: 0 10px 0 30px; width: 212px;"
                                            :class="{ active: isActive(child) }"
                                            @click="handleSelectChildRoute(child, routeIndex)"
                                        >
                                            <div class="route-info">
                                                <div class="route-icon">
                                                    <BaseSvg :name="child.meta.icon" :size="18"/>
                                                </div>
                                                <div class="route-title">
                                                    {{ $t(child.meta.title) }}
                                                </div>
                                            </div>
                                        </div>
                                    </li>
                                </ul>
                            </div>
                            <div
                                v-else
                                class="route-li"
                                :class="{ active: isActive(routeItem) }"
                                style="padding: 16px 20px;margin:0;width: 212px;"
                                @click="handleSelectRoute(routeItem, routeIndex)"
                            >
                                <div class="route-info">
                                    <div class="route-title">
                                        {{ $t(routeItem?.meta?.title) }}
                                    </div>
                                </div>
                            </div>
                        </template>
                        <div class="little-icon" :class="{ active: isLittleIconActive(routeItem) }" @click="handleClickLittleIcon(routeItem, routeIndex)">
                            <BaseSvg :name="routeItem.meta.icon" :size="18" />
                        </div>
                    </LaySidePopover>
                </li>
            </ul>
        </div>

        <!-- 展开收缩侧边按钮 -->
        <div class="fold-sidebar-container">
            <div class="fold-sidebar pointer" @mousemove="onMousemove" @click="appStore.manualToggleSidebar">
                <div 
                    v-show="state.hotZone.clientHeight >= state.hotZone.scrollHeight"
                    class="fold-sidebar-left"
                    :style="{width: !appStore.isCollapse ? '8px' : '4px'}" 
                />
                <div class="fold-sidebar-right">
                    <div class="fold-sidebar-right-trapezoid" :style="{ transform: `translate(-1px, ${state.mouse.y}px)` }">
                        <img v-if="!appStore.isCollapse" src="@/assets/images/trapezoid-left.png" alt="">
                        <img v-else src="@/assets/images/trapezoid-right.png" alt="">
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useAppStore } from '@/stores/modules/app'
import { computed, onBeforeMount, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { RouteRecordRaw, useRoute, useRouter } from 'vue-router'
import routerInstance from '@/router'
import LaySidePopover from './components/laySidePopover.vue'
import { hasPermission } from '@/utils/permission'

const route = useRoute()
const router = useRouter()

const appStore = useAppStore()

const scrollbarWrapperRef = ref<HTMLDivElement>(null)
const laySidePopoverRefs = ref<any[]>([])

const state = reactive({
    /** 菜单数据 */
    menu: [],
    /** 当前正在active的一级路由 */
    active: '',
    /** 当前正在active的二级路由 */
    activeChild: '',
    /** 是否已选择子路由 */
    isSelectChildRoute: false,
    /** 展开/收缩左侧hotZone(解决与滚动条冲突问题) */
    hotZone: {
        scrollHeight: 0,
        clientHeight: 0,
    },
    /** 侧边栏展开收缩按钮移动位置控制 */
    mouse: {
        x: 0,
        y: 0,
    },
})

/** 展开模式的按钮是否active */
const isActive = computed(() => {
    return (route: RouteRecordRaw) =>
        // 第一个父元素判断，第二个子元素判断
        (route.path === state.active && !route.meta?.whetherToExpandChildElements) || (route.path === state.activeChild)
})

/** 收缩模式的按钮是否active */
const isLittleIconActive = computed(() => {
    return (route: RouteRecordRaw) => {
        return route.path === state.active
    }
})

/** 选中一级路由 */
const handleSelectRoute = (route: RouteRecordRaw, routeIndex: number) => {
    if (route.children && route.children.length > 0) {
        // 手风琴模式
        if (!route.meta.whetherToExpandChildElements) {
            state.menu.forEach((item: RouteRecordRaw) => {
                item.meta.whetherToExpandChildElements = false
            })
        }
        // 正常切换状态
        route.meta.whetherToExpandChildElements = !route.meta.whetherToExpandChildElements
        return
    }
    router.push({ path: route.path })
    if (routeIndex || routeIndex === 0) {
        laySidePopoverRefs.value[routeIndex]?.close()
    }
}

/** 选中二级路由 */
const handleSelectChildRoute = (child: RouteRecordRaw, routeIndex: number) => {
    router.push({ path: child.path })
    if (routeIndex || routeIndex === 0) {
        laySidePopoverRefs.value[routeIndex].close()
    }
}

/** 点击收缩状态图标 */
const handleClickLittleIcon = (route: RouteRecordRaw, routeIndex: number) => {
    router.push({ path: route.path })
    if (routeIndex || routeIndex === 0) {
        laySidePopoverRefs.value[routeIndex].close()
    }
}

/** 侧边栏展开收缩按钮移动 */
const onMousemove = (e) => {
    if (e.clientY - 82 > -4) {
        if (e.clientY - 82 < window.innerHeight - 114) {
            state.mouse.y = e.clientY - 82
        } else {
            state.mouse.y = window.innerHeight - 114
        }
    } else {
        state.mouse.y = -4
    }
}

/** 改变窗口大小触发方法 */
const handleResize = () => {
    state.hotZone.clientHeight = scrollbarWrapperRef.value?.clientHeight
    state.hotZone.scrollHeight = scrollbarWrapperRef.value?.scrollHeight
    if (window.innerWidth < 1366) {
        appStore.autoCloseSidebar()
    } else {
        appStore.autoOpenSidebar()
    }
}

/** 将路由表的数据转化为menu需要的层级 */
const organizeRoutes = (routes: RouteRecordRaw[]) => {
    const routeMap = new Map()

    // 辅助函数：判断路由是否应该显示
    const shouldShowRoute = (route: RouteRecordRaw) => {
    // 没有 meta 字段，或者有 meta 但没有 menu 字段，都视为 false
        if (route.meta?.menu !== true) return false // 必须明确指定 menu: true 才显示
        // 如果声明了所需权限，当前用户必须具备该权限
        if (route.meta?.permission && !hasPermission(route.meta.permission)) return false
        return true
    }

    // 1. 预处理所有路由
    routes.forEach(route => {
    // 初始化 children 数组
        route.children = route.children || []
    
        // 过滤掉不应该显示的子路由
        route.children = route.children.filter(child => shouldShowRoute(child))
    
        // 初始化孙子路由
        route.children.forEach(child => {
            child.children = child.children || []
        })
    
        routeMap.set(route.path, route)
    })

    // 2. 筛选有效的一级路由（必须明确指定 menu: true）
    const validParentRoutes = routes.filter(route => {
        const isTopLevel = route.path.split('/').filter(Boolean).length === 1
        return isTopLevel && shouldShowRoute(route)
    })

    // 3. 递归处理子路由
    function processChildren (parentRoute: RouteRecordRaw) {
        routes.forEach(route => {
            const parentPath = parentRoute.path === '/' ? '' : parentRoute.path
            const pathSegments = route.path.replace(/\/$/, '').split('/')
            const parentSegments = parentRoute.path.replace(/\/$/, '').split('/')
      
            // 判断是否是直接子路由
            const isDirectChild = 
        route.path.startsWith(`${parentPath}/`) && 
        pathSegments.length === parentSegments.length + 1

            // 满足三个条件才添加：
            // 1. 是直接子路由
            // 2. 应该显示（menu: true）
            // 3. 不在现有 children 中（避免重复）
            if (isDirectChild && shouldShowRoute(route) && 
          !parentRoute.children.some(child => child.path === route.path)) {
                processChildren(route)
                parentRoute.children.push(route)
            }
        })
    }

    // 处理有效路由树
    validParentRoutes.forEach(parentRoute => {
        processChildren(parentRoute)
        // 处理初始 children 的多级嵌套
        parentRoute.children.forEach(child => processChildren(child))
    })

    return validParentRoutes
}

interface RouteLevelInfo {
  level: number;
  parent: RouteRecordRaw | null;
}

/**
 * 递归查找路由路径的层级和父级
 * @param routes 路由列表
 * @param path 要查找的路径
 * @param parent 父级路由(内部递归使用)
 * @param currentLevel 当前层级(内部递归使用)
 * @returns RouteLevelInfo 包含层级和父级的信息
 */
const findRouteLevel = (
    routes: RouteRecordRaw[],
    path: string,
    parent: RouteRecordRaw | null = null,
    currentLevel: number = 1,
): RouteLevelInfo => {
    for (const route of routes) {
        if (route.path === path) {
            return {
                level: currentLevel,
                parent: parent,
            }
        }

        if (route.children && route.children.length > 0) {
            const found = findRouteLevel(route.children, path, route, currentLevel + 1)
            if (found.level !== -1) {
                return found
            }
        }
    }

    // 如果没有找到
    return {
        level: -1,
        parent: null,
    }
}

// 计算高度
const calculateHeight = (num: number) => {
    return (num * 48) + 'px'
}

onBeforeMount(() => {
    watch(() => route, (val) => {
        const { level, parent } = findRouteLevel(state.menu, val.path)
        if (level === 2) {
            state.isSelectChildRoute = true
            state.active = parent.path
            state.activeChild = val.path
        } else {
            state.isSelectChildRoute = false
            // 判断是否有父级联动选中
            state.active = val.meta?.linkageParentLevelPath ? val.meta?.linkageParentLevelPath : val.path
            state.activeChild = undefined
        }
    
    }, { immediate: true, deep: true })

    // 添加监听窗口改变
    window.addEventListener('resize', handleResize)
    handleResize()
})

onBeforeUnmount(() => {
    // 移除监听窗口改变
    window.removeEventListener('resize', handleResize)
})

/** 初始化 */
const init = () => {
    // handleResize()
    const routerInstanceList = routerInstance.options.routes.find(item => item.path === '/')
    state.menu = organizeRoutes(routerInstanceList.children)

    // 如果当前路由在二级菜单中，则展开父级菜单
    const routeRes = findRouteLevel(state.menu, route.path)
    if (routeRes && routeRes.level === 2) {
        state.menu.find(item => item.path === routeRes.parent.path).meta.whetherToExpandChildElements = true
    }
}

init()
</script>

<style scoped lang="scss">
ul {
    list-style: none;
    padding: 0;
    margin: 0;
}

.scrollbar-container {
    position: relative;

    // 菜单主体-展开
    .scrollbar-wrapper {
        height: 100%;
        overflow-y: hidden;
        overflow-x: visible;
        background-color: var(--gl-color-bg-sidebar);
        width: 232px;
        padding: 24px 10px 52px;
        transition: width 0.3s;

        &:hover {
            overflow-y: auto;
        }

        .menu-list {
            user-select: none;
        }

        .little-icon-list {
            display: flex;
            justify-content: center;
            // align-items: center;
            flex-direction: column;
        }
    }

    // 菜单主体-收缩
    .scrollbar-wrapper-collapsed {
        width: 56px;
        padding: 4px 4px 32px;
    }

    // 展开收缩侧边按钮
    .fold-sidebar-container {
        position: absolute;
        right: -17px;
        top: 0;

        height: 100%;

        &:hover {
            .fold-sidebar-right-trapezoid {
                display: block !important;
            }
        }

        .fold-sidebar {
            display: flex;
            height: 100%;
            

            .fold-sidebar-left {
                width: 8px;
                height: 100%;
            }
            .fold-sidebar-right {
                width: 16px;
                height: 100%;
                background-color: transparent;
                position: relative;

                .fold-sidebar-right-trapezoid {
                    width: 16px;

                    img {
                        width: 100%;
                        height: 100%;
                        display: block;
                    }

                    display: none;

                    position: absolute;
                    top: 0;
                    left: 0;
                }
            }
        }
    }
}

.route-li {
    width: 212px;
    height: 40px;
    background-color: transparent;
    color: var(--gl-color-text-level2);
    border-radius: 20px;
    padding: 0 16px;
    margin-top: 8px;
    cursor: pointer;

    display: flex;
    justify-content: space-between;
    align-items: center;

    transition: all 0.2s;

    .route-info {
        display: flex;
        align-items: center;

        .route-icon {
            margin-right: 6px;
            width: 18px;
            height: 18px;
            line-height: 18px;
        }

        .route-title {
            font-size: 14px;
        }
    }
    .chevron-down-icon {
        display: flex;
        justify-content: center;
        align-items: center;
        transform: rotate(0deg);
        transition: all 0.3s;
        .gl-icon {
            font-size: 12px !important;
        }
    }
    .icon-active {
        transform: rotate(180deg);
        transition: all 0.3s;
    }

    &:hover {
        background-color: var(--gl-color-bg-icon-button-hover);
    }

    &.active {
        background-color: var(--gl-color-brand-background);
        color: var(--gl-color-brand-primary);
    }
}

.child-route-list {
    transition: max-height 0.4s;
    -webkit-transition: max-height 0.4s;
    // overflow-x: hidden;
    overflow: hidden;
    min-width: 212px;
}

.route-li-enter-active, .route-li-leave-active {
    transition: all 0.4s;
}
.route-li-enter {
    opacity: 0;
}
.route-li-leave-to {
    opacity: 0;
}

 .little-title {
    width: 100%;
    padding: 16px;
    border-bottom: 1px solid var(--gl-color-line-divider1);
    color: var(--gl-color-text-level2);
    font-size: 14px;
}

.little-icon {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    color: var(--gl-color-text-level2);
    margin-top: 8px;
    margin-left: 4px;

    display: flex;
    justify-content: center;
    align-items: center;

    cursor: pointer;

    transition: all 0.2s;

    &:hover {
        background-color: var(--gl-color-bg-icon-button-hover);
    }

    &.active {
        background-color: var(--gl-color-brand-background);
        color: var(--gl-color-brand-primary);
    }
}
</style>