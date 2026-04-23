<!--
 * @Description: 编辑个人信息（仅 displayName）
-->
<template>
    <BaseModal
        :width="520"
        :open="props.open"
        :title="$t('personalCenter.edit')"
        destroyOnClose
        :beforeOk="handleApply"
        @close="emits('update:open', false)"
    >
        <AForm
            ref="formRef"
            :colon="false"
            :model="state.formData"
            :rules="formRules"
            :labelCol="{ span: 8 }"
            :wrapperCol="{ span: 16 }"
        >
            <AFormItem name="displayName" :label="$t('personalCenter.displayName')" labelAlign="left">
                <ATextarea
                    v-model:value="state.formData.displayName"
                    :maxlength="200"
                    :placeholder="$t('common.maxLength', { length: 200 })"
                    style="width: 100%;"
                />
            </AFormItem>
        </AForm>
    </BaseModal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import BaseModal from '@/components/base/baseModalI18n.vue'
import { FormRules, OnBeforeOk } from 'gl-web-main'
import { FormInstance, message } from 'ant-design-vue'
import { reqUpdateProfile } from '@/api/personal'
import { PersonalProfile } from '@/models/personal'
import { t } from '@/hooks/useLanguage'

const props = defineProps<{ open: boolean, profile: PersonalProfile | null }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void
    (e: 'handleApply'): void
}>()

const formRef = ref<FormInstance>()

const state = reactive({
    formData: {
        displayName: '' as string,
    },
})

const formRules: FormRules = {
    displayName: [
        { max: 200, message: t('common.maxLength', { length: 200 }), trigger: 'change' },
    ],
}

watch(() => props.open, (open) => {
    if (open) {
        state.formData.displayName = props.profile?.displayName || ''
    }
})

const handleApply: OnBeforeOk = (done) => {
    formRef.value.validate().then(() => {
        reqUpdateProfile({ displayName: state.formData.displayName || '' })
            .then(() => {
                message.success(t('common.success'))
                emits('handleApply')
                done(true)
            })
            .catch(() => done(false))
    }).catch(() => done(false))
}
</script>
