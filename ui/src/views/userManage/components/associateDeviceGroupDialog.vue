<!--
 * @Author: LPY
 * @Date: 2026-02-03 14:37:09
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-03 14:56:28
 * @FilePath: \glkvm-cloud\ui\src\views\userManage\components\associateDeviceGroupDialog.vue
 * @Description: 关联设备组弹窗
-->
<template>
    <BaseModal
        :width="500"
        :open="props.open"
        :title="$t('user.associatedDeviceGroup')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <div>
            <BaseInfo style="margin-bottom: 32px;">
                <div class="flex-start flex-nowrap">
                    <BaseSvg name="gl-icon-info-circle" :size="24" style="color: var(--gl-color-brand-primary); margin-right: 10px;"></BaseSvg>
                    <BaseText>{{ $t('user.associatedDeviceGroupTips') }}</BaseText>
                </div>
            </BaseInfo>

            <AForm
                :colon="false"
                :rules="formRules"
                :model="state.formData"
                ref="formRef"
            >
                <AFormItem name="groupId" :label="$t('device.deviceGroup')" :labelCol="{ span: 8 }" :wrapperCol="{ span: 16 }" labelAlign="left">
                    <ASelect v-model:value="state.formData.deviceGroupIds" mode="multiple" :placeholder="$t('common.pleaseSelect')" style="width: 100%;">
                        <ASelectOption v-for="item in state.groupList" :key="item.groupId" :value="item.groupId">{{ item.name }}</ASelectOption>
                    </ASelect>
                </AFormItem>
            </AForm>
        </div>
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { BaseInfo, BaseModal } from 'gl-web-main/components'
import { FormRules, OnBeforeOk } from 'gl-web-main'
import { t } from '@/hooks/useLanguage'
import { FormInstance } from 'ant-design-vue'
import { reqDeviceGroupListOptions } from '@/api/device'
import { UserGroup } from '@/models/userManage'
import { reqAssociateDeviceGroup } from '@/api/userManage'

const props = defineProps<{ open: boolean, currentUserGroup: UserGroup | null }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
    (e: 'handleApply'): void;
}>()

const formRef = ref<FormInstance>()

const state = reactive({
    groupList: [],
    formData: {
        deviceGroupIds: undefined,
    },
})

/** 表单验证 */
const formRules: FormRules = {
    deviceGroupIds: [
        { required: true, message: t('device.requiredDeviceGroup'), trigger: 'change' },
    ],
}

const getDeviceGroupListOptions = async () => {
    const res = await reqDeviceGroupListOptions()
    state.groupList = res.data.items
}

/** 提交 */
const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        reqAssociateDeviceGroup(props.currentUserGroup?.id, { deviceGroupIds: state.formData.deviceGroupIds }).then(() => {
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

const init = () => {
    state.formData.deviceGroupIds = props.currentUserGroup?.deviceGroupList.map(item => item.deviceGroupId)
    getDeviceGroupListOptions()
}
</script>

<style lang="scss" scoped>

</style>