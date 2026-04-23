<!--
 * @Author: LPY
 * @Date: 2026-02-03 14:29:50
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-03 14:38:27
 * @FilePath: \glkvm-cloud\ui\src\views\userManage\components\editUserGroupDialog.vue
 * @Description: 编辑用户组弹窗
-->
<template>
    <BaseModal
        :width="500"
        :open="props.open"
        :title="$t('user.editUserGroup')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <div class="add-user-group-container">
            <AForm
                :colon="false"
                :rules="formRules"
                :model="state.formData"
                :labelCol="{ span: 8 }"
                :wrapperCol="{ span: 16 }"
                ref="formRef"
            >
                <AFormItem name="name" :label="$t('user.userGroupName')"   labelAlign="left">
                    <AInput v-model:value="state.formData.name" :maxlength="32" :placeholder="$t('device.requiredDeviceGroupName')" style="width: 100%;" />
                </AFormItem>
                <AFormItem name="description" :label="$t('device.description')" labelAlign="left">
                    <ATextarea 
                        v-model:value="state.formData.description"
                        :maxlength="200"
                        :placeholder="$t('device.requiredDeviceGroupDescription')"
                        style="width: 100%;" />
                </AFormItem>
            </AForm>
        </div>
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import BaseModal from '@/components/base/baseModalI18n.vue'
import { FormRules, OnBeforeOk } from 'gl-web-main'
import { t } from '@/hooks/useLanguage'
import { FormInstance, message } from 'ant-design-vue'
import { reqEditUserGroup } from '@/api/userManage'
import { UserGroup } from '@/models/userManage'

const props = defineProps<{ open: boolean, currentUserGroup: UserGroup | null }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply', value: number): void;
}>()

const formRef = ref<FormInstance>()

const state = reactive({
    formData: {
        name: undefined,
        description: undefined,
        userGroupIds: [],
    },
})

/** 表单验证 */
const formRules: FormRules = {
    name: [
        { required: true, message: t('device.requiredDeviceGroupName'), trigger: 'change' },
    ],
}

/** 提交 */
const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        reqEditUserGroup(props.currentUserGroup?.id, { 
            name: state.formData.name,
            description: state.formData.description,
        }).then((res) => {
            emits('handleApply', res.data.id)
            message.success(t('common.success'))
            done(true)
        }).catch(() => {
            done(false)
        })
    }).catch(() => {
        done(false)
    })
}

/** 初始化数据 */
watch(() => props.open, (newVal) => {
    if (newVal) {
        init()
    }
})

const init = async () => {
    state.formData.name = props.currentUserGroup?.userGroup
    state.formData.description = props.currentUserGroup?.description
}
</script>

<style lang="scss" scoped>

</style>