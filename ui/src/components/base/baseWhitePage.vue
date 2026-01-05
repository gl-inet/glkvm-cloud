<!--
 * @Author: LPY
 * @Date: 2025-06-09 09:29:48
 * @LastEditors: LPY
 * @LastEditTime: 2026-01-05 14:30:56
 * @FilePath: \glkvm-cloud\ui\src\components\base\baseWhitePage.vue
 * @Description: 基础白名单页。
-->
<template>
    <div class="base-white-page">
        <div class="base-white-page-header">
            <div class="base-white-page-header-left">
                <img src="@/assets/svg/logo.svg" height="20">
            </div>
            <div class="base-white-page-header-right">
                <BaseText style="margin-right: 24px;">{{ appStore.state.version || '--' }}</BaseText>
                <BaseDropdownSelect :value="currentLang" :options="languageOptions" @update:value="changeLang">
                    <div class="language-box flex">
                        <BaseSvg name="gl-icon-language-regular" style="margin-right: 8px;font-size: 16px;"></BaseSvg>
                        {{ currentLangLabel }}
                        <BaseSvg name="gl-icon-chevron-down-regular" style="margin-left: 8px;font-size: 16px;"></BaseSvg>
                    </div>
                </BaseDropdownSelect>
            </div>
        </div>

        <div class="base-white-page-footer">
            <!-- 步骤条 -->
            <!-- <div v-if="route.query.bindToken" class="base-white-page-step">
                <BaseStep 
                    v-model:value="useUserStore().bindingStep"
                    :items="[{title: $t('login.accountSetup')}, {title: $t('login.deviceSetup')}]"
                    titlePosition="bottom"
                />
            </div> -->

            <div class="base-white-page-content">
                <slot></slot>
            </div>

            <!-- 版权信息 -->
            <div class="copyright-info">
                Copyright © 2025 GL.iNet. All Rights Reserved
            </div>

            <!-- 国内备案号信息 -->
            <div v-if="!isForeignEnv" class="filing-info">
                <a href="https://beian.miit.gov.cn/" target="_blank" rel="noopener noreferrer" style="color: var(--gl-color-text-level3);">粤ICP备18130956号-9</a>
                深圳市广联智通科技有限公司
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import useLanguage from '@/hooks/useLanguage'
import { languageOptions, Languages } from 'gl-web-main'
import { BaseDropdownSelect } from 'gl-web-main/components'
import { isForeignEnv } from '@/utils'
import { useAppStore } from '@/stores/modules/app'

const { currentLang, currentLangLabel } = useLanguage()

const changeLang = (key: Languages) => {
    useLanguage().setLanguage(key)
}

const appStore = useAppStore()
</script>

<style scoped lang="scss">
.base-white-page {
  height: 100vh;
  .base-white-page-header {
    height: 44px;
    padding: 0 20px;

    display: flex;
    justify-content: space-between;
    align-items: center;

    background-color: var(--gl-color-bg-surface1);

    border-bottom: 1px solid var(--gl-color-line-divider1);

    .base-white-page-header-right {
      display: flex;
      align-items: center;
      .language-box {
        color: var(--gl-color-text-level2);
        user-select: none;
      }
    }
  }

  .base-white-page-footer {
    background-color: var(--gl-color-bg-page);
    height: calc(100% - 44px);

    display: flex;
    flex-direction: column;
    justify-content: space-between;
    align-items: center;

    padding-bottom: 30px;

    overflow: auto;

    .base-white-page-step {
        width: 314px;
        padding-bottom: 28px;
        margin-top: 36px;
    }

    .base-white-page-content {
      flex: 1;

      display: flex;
      justify-content: center;
      align-items: center;
    }

    .copyright-info {
      color: var(--gl-color-text-level3);
    }

    .filing-info {
      color: var(--gl-color-text-level3);
      margin-top: 6px;
    }
  }
}
</style>