<!--
 * @Author: LPY
 * @Date: 2026-02-02 14:34:32
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 11:09:14
 * @FilePath: \glkvm-cloud\ui\src\views\userManage\userPage.vue
 * @Description: 用户页
-->
<template>
    <div class="user-container">
        <ATabs v-if="isAdmin" v-model:activeKey="activeKey" style="height: 100%">
            <ATabPane key="1" :tab="$t('user.user') + '(' + userManageStore.state.userList.length + ')'" style="height: 100%;">
                <UserManagePage />
            </ATabPane>
            <ATabPane key="2" :tab="$t('user.userGroup') + '(' + userGroupManageStore.state.userGroupList.length + ')'" forceRender style="height: 100%;">
                <UserGroupManagePage />
            </ATabPane>
        </ATabs>
        <UserGroupManagePage v-else />
    </div>

</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import UserManagePage from './userManagePage.vue'
import UserGroupManagePage from './userGroupManagePage.vue'
import { useUserManageStore } from '@/stores/modules/userManage'
import { useUserGroupManageStore } from '@/stores/modules/userGroupManage'
import { hasPermission } from '@/utils/permission'
import { PermissionEnum } from '@/models/permission'
const activeKey = ref('1')
const userManageStore = useUserManageStore()
const userGroupManageStore = useUserGroupManageStore()

const isAdmin = computed(() => {
    return hasPermission(PermissionEnum.USER_WRITE)
})
</script>

<style scoped lang="scss">
:deep(.ant-tabs-tab) {
  font-size: 24px !important;
  font-weight: 500;
  padding: 0 0 10px 0 !important;
}
:deep(.ant-tabs-content) {
  height: 100%;
}
.user-container {
  height: 100%;
  padding: 20px 24px;
  background-color: var(--gl-color-bg-page);
}
</style>