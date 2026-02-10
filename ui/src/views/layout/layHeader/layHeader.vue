<!--
 * @Author: LPY
 * @Date: 2025-05-30 15:21:14
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-06 18:00:59
 * @FilePath: \glkvm-cloud\ui\src\views\layout\layHeader\layHeader.vue
 * @Description: 顶部集成页
-->
<template>
    <div class="lay-header">
        <div class="lay-header-left">
            <img src="@/assets/svg/logo.svg" height="20">
        </div>
        <div class="lay-header-right">
            <!-- version -->
            <BaseText style="margin-right: 24px;">{{ appStore.state.version || '--' }}</BaseText>
            <!-- github -->
            <ATooltip>
                <template #title>{{ githubLink }}</template>
                <a :href="githubLink" target="_blank" rel="noopener noreferrer" class="icon-area">
                    <BaseSvg name="gl-icon-github" :size="24" />
                </a>
            </ATooltip>
            <!-- 问题指引 -->
            <ATooltip>
                <template #title>{{ helpLink }}</template>
                <a :href="helpLink" target="_blank" rel="noopener noreferrer" class="icon-area">
                    <BaseSvg name="gl-icon-help" :size="24" />
                </a>
            </ATooltip>
            <!-- 竖线 -->
            <div class="vertical-line" />

            <ADropdown :trigger="['click']" overlayClassName="base-dropdown header-dropdown">
                <div class="dropdown-box">
                    <!-- 用户头像 -->
                    <div class="user-avatar">
                        {{ getUserAvatarInitials(userStore.userInfo?.user?.username) }}
                    </div>
                    <!-- 用户名 -->
                    <BaseText type="body-r" variant="level2" style="margin: 0 4px;">
                        {{ userStore.userInfo?.user?.username }}
                    </BaseText>
                    <!-- 下拉箭头 -->
                    <BaseSvg name="gl-icon-chevron-down-regular" style="font-size: 16px;color: var(--gl-color-text-level3);margin: 0 4px;" />
                </div>

                <template #overlay>
                    <a-menu>
                        <div class="user-info-dropdown-box">
                            <div class="user-info-dropdown-avatar">
                                {{ getUserAvatarInitials(userStore.userInfo?.user?.username) }}
                            </div>
                            <BaseText type="body-r" style="margin: 30px 0 4px;">
                                {{ userStore.userInfo?.user?.username }}
                            </BaseText>
                            <BaseText type="body-r" variant="level2" style="margin-bottom: 2px;">
                                <BaseTag v-if="userStore.userInfo?.user?.role === UserRoleEnum.USER" primary>
                                    {{ $t(UserRoleLabelMap.get(userStore.userInfo?.user?.role)) }}
                                </BaseTag>
                                <BaseTag 
                                    v-else-if="userStore.userInfo?.user?.role === UserRoleEnum.ADMIN"
                                    style="background-color: var(--gl-color-warning-primary);color: var(--gl-color-warning-background);"
                                >{{ $t(UserRoleLabelMap.get(userStore.userInfo?.user?.role)) }}</BaseTag>
                            </BaseText>
                        </div>
                        <div class="user-info-divider-line" />
                        <a-menu-item class="dropdown-menu-item-danger" @click="userStore.manualLogout">
                            <a>{{ $t('login.signOut') }}</a>
                        </a-menu-item>
                    </a-menu>
                </template>
            </ADropdown>
        </div>
    </div>
</template>

<script setup lang="ts">
import { UserRoleEnum, UserRoleLabelMap } from '@/models/userManage'
import { useAppStore } from '@/stores/modules/app'
import { useUserStore } from '@/stores/modules/user'
import { getUserAvatarInitials } from '@/utils/user'
import { BaseTag } from 'gl-web-main/components'

const userStore = useUserStore()
const appStore = useAppStore()

// github链接
const githubLink = 'https://github.com/gl-inet/glkvm-cloud'

// 问题指引链接
const helpLink = 'https://www.gl-inet.com'
</script>

<style scoped lang="scss">
.lay-header {
  width: 100%;
  height: 56px;
  line-height: 56px;
  border-bottom: 1px solid var(--gl-color-line-divider1);

  display: flex;
  justify-content: space-between;
  align-items: center;

  padding: 8px 20px;

  .lay-header-left {
    height: 100%;

    display: flex;
    align-items: center;
  }

  .lay-header-right {
    height: 100%;

    display: flex;
    align-items: center;

    .icon-area {
      width: 36px;
      height: 36px;
      display: flex;
      justify-content: center;
      align-items: center;
      border-radius: 4px;
      color: var(--gl-color-text-level3);
      cursor: pointer;

      &:hover {
        background-color: var(--gl-color-bg-icon-button-hover);
      }
    }

    .vertical-line {
      width: 1px;
      height: 24px;
      background-color: var(--gl-color-line-divider1);
      margin: 0 12px;
    }

    .log-out {
      display: flex;
      align-items: center;
      color: var(--gl-color-text-level3);
    }
    
    .dropdown-box {
      display: flex;
      align-items: center;
      height: 40px;
      border-radius: 20px;
      user-select: none;
      cursor: pointer;

      &:hover {
        background-color: var(--gl-color-bg-icon-button-hover);
      }
      
      .user-avatar {
        width: 32px;
        height: 32px;
        border-radius: 50%;
        background-color: var(--gl-color-brand-background);
        font-size: 16px;
        color: var(--gl-color-brand-primary);
        margin: 0 4px;
  
        display: flex;
        justify-content: center;
        align-items: center;
  
        user-select: none;
      }
    }
  }
}

.header-dropdown {
  .ant-dropdown-menu {
    .user-info-dropdown-box {
      width: 212px;
      margin-top: 22px;
      display: flex;
      flex-direction: column;
      align-items: center;
      .user-info-dropdown-avatar {
        width: 80px;
        height: 80px;
        border-radius: 50%;
        border: 1px solid var(--gl-color-brand-primary);
        display: flex;
        justify-content: center;
        align-items: center;
        background-color: var(--gl-color-brand-background);
        color: var(--gl-color-brand-primary);
        font-size: 24px;
        user-select: none;
      }
    }
    
    .user-info-divider-line {
      width: 100%;
      height: 1px;
      background-color: var(--gl-color-line-divider2);
      margin: 8px 0;
    }
  }
}
</style>