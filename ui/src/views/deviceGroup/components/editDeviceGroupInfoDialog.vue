<!--
 * @Author: LPY
 * @Date: 2026-02-02 12:27:10
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 11:57:07
 * @FilePath: \glkvm-cloud\ui\src\views\deviceGroup\components\editDeviceGroupInfoDialog.vue
 * @Description: 编辑设备组信息弹窗
-->
<template>
    <BaseModal
        :width="626"
        :open="props.open"
        :title="$t('device.editBasicInfo')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <div class="edit-device-group-info-container">
            <div class="form">
                <AForm
                    :colon="false"
                    :rules="formRules"
                    :model="state.formData"
                    :labelCol="{ span: 8 }"
                    :wrapperCol="{ span: 16 }"
                    ref="formRef"
                >
                    <AFormItem name="name" :label="$t('device.deviceGroupName')"   labelAlign="left">
                        <AInput v-model:value="state.formData.name" :maxlength="32" :placeholder="$t('device.requiredDeviceGroupName')" style="width: 100%;" />
                    </AFormItem>
                    <AFormItem name="description" :label="$t('device.description')" labelAlign="left">
                        <ATextarea 
                            v-model:value="state.formData.description"
                            :maxlength="200"
                            :placeholder="$t('device.requiredDeviceGroupDescription')"
                            style="width: 100%;" />
                    </AFormItem>
                    <AFormItem name="userGroupIds" :label="$t('device.associatedUserGroups')" labelAlign="left">
                        <ASelect v-model:value="state.formData.userGroupIds" mode="multiple" :placeholder="$t('common.pleaseSelect')" style="width: 100%;">
                            <ASelectOption v-for="item in state.userGroupList" :key="item.userGroupId" :value="item.userGroupId">{{ item.name }}</ASelectOption>
                        </ASelect>
                    </AFormItem>
                </AForm>
            </div>
        </div>
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { BaseModal } from 'gl-web-main/components'
import { FormRules, OnBeforeOk } from 'gl-web-main'
import { t } from '@/hooks/useLanguage'
import { FormInstance } from 'ant-design-vue'
import { reqEditDeviceGroup, reqUserGroupListOptions } from '@/api/deviceGroup'
import { DeviceGroup } from '@/models/deviceGroup'

const props = defineProps<{ open: boolean, currentDeviceGroup: DeviceGroup | null }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply'): void;
}>()


const formRef = ref<FormInstance>()

const state = reactive({
    formData: {
        name: undefined,
        description: undefined,
        userGroupIds: [],
    },
    userGroupList: [],
})

/** 表单验证 */
const formRules: FormRules = {
    name: [
        { required: true, message: t('device.requiredDeviceGroupName'), trigger: 'change' },
    ],
}

/** 获取用户组下拉选项 */
const getUserGroupListOptions = async () => {
    const res = await reqUserGroupListOptions()
    state.userGroupList = res.data.items
}

/** 提交 */
const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        reqEditDeviceGroup(props.currentDeviceGroup?.id, { 
            name: state.formData.name,
            description: state.formData.description,
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
    state.formData.name = props.currentDeviceGroup?.name
    state.formData.description = props.currentDeviceGroup?.description
    state.formData.userGroupIds = props.currentDeviceGroup?.userGroupList.map(item => item.userGroupId)
    getUserGroupListOptions()
}
</script>

<style lang="scss" scoped>

</style>