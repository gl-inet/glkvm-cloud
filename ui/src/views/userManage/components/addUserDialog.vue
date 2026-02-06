<!--
 * @Author: LPY
 * @Date: 2026-02-03 10:02:21
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-06 14:04:03
 * @FilePath: \glkvm-cloud\ui\src\views\userManage\components\addUserDialog.vue
 * @Description: 添加用户弹窗
-->
<template>
    <BaseModal
        :width="626"
        :open="props.open"
        :title="$t('user.addUser')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <div class="add-user-container">
            <AForm
                :colon="false"
                :rules="formRules"
                :model="state.formData"
                :labelCol="{ span: 8 }"
                :wrapperCol="{ span: 16 }"
                ref="formRef"
            >
                <AFormItem name="role" labelAlign="left">
                    <template #label>
                        {{ $t('user.userRole') }}
                        <BaseSvg name="gl-icon-help" :size="18" style="margin-left: 8px;" tooltip>
                            {{ $t('user.userRoleDesc') }}
                        </BaseSvg>
                    </template>
                    <ARadioGroup v-model:value="state.formData.role" name="role">
                        <ARadio :value="UserRoleEnum.USER">{{ UserRoleLabelMap.get(UserRoleEnum.USER) }}</ARadio>
                        <ARadio :value="UserRoleEnum.ADMIN">{{ UserRoleLabelMap.get(UserRoleEnum.ADMIN) }}</ARadio>
                    </ARadioGroup>
                </AFormItem>
                <AFormItem name="username" :label="$t('user.userName')"   labelAlign="left">
                    <AInput v-model:value="state.formData.username" :maxlength="32" :placeholder="$t('device.requiredDeviceGroupName')" style="width: 100%;" />
                </AFormItem>
                <AFormItem name="description" :label="$t('device.description')" labelAlign="left">
                    <ATextarea 
                        v-model:value="state.formData.description"
                        :maxlength="200"
                        :placeholder="$t('device.requiredDeviceGroupDescription')"
                        style="width: 100%;" />
                </AFormItem>
                <AFormItem name="password" :label="$t('user.setPassword')"   labelAlign="left">
                    <AInputPassword 
                        v-model:value="state.formData.password"
                        :placeholder="$t('user.enterPassword')"
                        autocomplete="off"
                        style="width: 100%;" />
                </AFormItem>
                <AFormItem name="repassword" :label="$t('user.reEnterPassword')"   labelAlign="left">
                    <AInputPassword
                        v-model:value="state.formData.repassword"
                        :placeholder="$t('user.reEnterPasswordPlc')"
                        autocomplete="off"
                        style="width: 100%;" />
                </AFormItem>
                <div class="flex-end">
                    <a @click="state.addUserGroupDialogOpen = true">{{ $t('device.notFoundDeviceGroup') }}</a>
                </div>
                <AFormItem name="userGroupIds" :label="$t('device.associatedUserGroups')" labelAlign="left">
                    <ASelect v-model:value="state.formData.userGroupIds" mode="multiple" :placeholder="$t('common.pleaseSelect')" style="width: 100%;">
                        <ASelectOption v-for="item in state.userGroupList" :key="item.userGroupId" :value="item.userGroupId">{{ item.name }}</ASelectOption>
                    </ASelect>
                </AFormItem>
            </AForm>
        </div>

        <!-- 添加用户组弹窗 -->
        <AddUserGroupDialog 
            v-model:open="state.addUserGroupDialogOpen"
            @handleApply="addUserGroupApply"
        />
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { BaseModal } from 'gl-web-main/components'
import { FormRules, OnBeforeOk } from 'gl-web-main'
import { t } from '@/hooks/useLanguage'
import { FormInstance } from 'ant-design-vue'
import { reqUserGroupListOptions } from '@/api/deviceGroup'
import { UserRoleEnum, UserRoleLabelMap } from '@/models/userManage'
import { reqCreateUser } from '@/api/userManage'
import AddUserGroupDialog from './addUserGroupDialog.vue'

const props = defineProps<{ open: boolean }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply'): void;
}>()

const formRef = ref<FormInstance>()

const state = reactive({
    formData: {
        role: UserRoleEnum.USER,
        username: undefined,
        description: undefined,
        password: undefined,
        repassword: undefined,
        userGroupIds: [],
    },
    userGroupList: [],
    addUserGroupDialogOpen: false,
})

/** 表单验证 */
const formRules: FormRules = {
    username: [
        { required: true, message: t('device.requiredDeviceGroupName'), trigger: 'change' },
    ],
    password: [
        { required: true, message: t('user.enterPassword'), trigger: 'change' },
    ],
    repassword: [{
        required: true,
        trigger: 'blur',
        validator: async (_: any, value: string) =>  {
            if (value && value === state.formData.password) {
                return Promise.resolve()
            }
            return Promise.reject(t('login.confirmPasswordValidateError'))
        },
    }],
}

/** 获取用户组下拉选项 */
const getUserGroupListOptions = async () => {
    const res = await reqUserGroupListOptions()
    state.userGroupList = res.data.items
}

/** 添加用户组弹窗成功回调 */
const addUserGroupApply = async (id: number) => {
    await getUserGroupListOptions()
    state.formData.userGroupIds.push(id)
}

/** 提交 */
const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        reqCreateUser({
            role: state.formData.role,
            username: state.formData.username,
            description: state.formData.description,
            password: state.formData.password,
            repassword: state.formData.repassword,
            userGroupIds: state.formData.userGroupIds,
        }).then(() => {
            emits('handleApply')
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
    state.formData.role = UserRoleEnum.USER
    state.formData.username = undefined
    state.formData.description = undefined
    state.formData.password = undefined
    state.formData.repassword = undefined
    state.formData.userGroupIds = []
    getUserGroupListOptions()
}
</script>

<style lang="scss" scoped>

</style>