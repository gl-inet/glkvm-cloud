<!--
 * @Description: 个人中心页（Personal Center）
-->
<template>
    <div class="personal-center-container">
        <BaseLoadingContainer :spinning="state.loading">
            <PersonalInformation
                :profile="state.profile"
                @refresh="loadProfile"
            />
            <SecuritySettings
                :profile="state.profile"
                @refresh="loadProfile"
            />
        </BaseLoadingContainer>
    </div>
</template>

<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import BaseLoadingContainer from '@/components/base/baseLoadingContainer.vue'
import PersonalInformation from './components/personalInformation.vue'
import SecuritySettings from './components/securitySettings.vue'
import { reqGetProfile } from '@/api/personal'
import { PersonalProfile } from '@/models/personal'

const state = reactive({
    loading: false,
    profile: null as PersonalProfile | null,
})

const loadProfile = async () => {
    state.loading = true
    try {
        const res = await reqGetProfile()
        state.profile = res.data
    } finally {
        state.loading = false
    }
}

onMounted(loadProfile)
</script>

<style scoped lang="scss">
.personal-center-container {
  height: 100%;
  padding: 20px 24px;
  background-color: var(--gl-color-bg-page);
  overflow-y: auto;

  > :deep(.ant-spin-nested-loading) > .ant-spin-container {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
}
</style>
