<!--
 * @Author: LPY
 * @Date: 2025-05-30 10:48:43
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-28 09:32:36
 * @FilePath: \glkvm-cloud\ui\src\views\login\loginPage.vue
 * @Description: 登录页面
-->
<template>
    <BaseWhitePage>
        <LoginBox>
            <div style="display: flex; align-items: center; justify-content: center; margin: 24px 0 16px;">
                <BaseText type="large-title-m">
                    {{ $t('login.authorizationRequired') }}
                </BaseText>
            </div>

            <!-- 切换登录方式 -->
            <BaseRadioButtonsCompact
                v-if="isLdapEnabled"
                v-model:value="state.formModel.authMethod"
                :options="LoginTypeOptionsTranslated.value"
                style="width: 100%; margin-bottom: 16px;"
            />

            <AForm 
                class="dense-form" 
                ref="formRef"
                :model="state.formModel"
                :rules="formRules"
                :validateTrigger="['blur', 'change']"
                style="width: 100%;"
                @validate="handleValidate"
            >
                <!-- 用户名字段 -->
                <AFormItem name="username">
                    <GlInput
                        name="username"
                        v-model:value="state.formModel.username"
                        :placeholder="$t('login.username')"
                        @pressEnter="handleLogin"
                    />
                </AFormItem>

                <AFormItem name="password">
                    <GlPassword
                        name="password"
                        :useDefaultValidateRule="false"
                        @pressEnter="handleLogin"
                        v-model:value="state.formModel.password"
                        :placeholder="$t('login.password')"
                    />
                </AFormItem>
            </AForm>
            <BaseButton medium type="primary" style="width: 100%;margin-top: 16px;" :loading="state.loading" @click="handleLogin">
                {{ $t('login.signIn') }}
            </BaseButton>

            <div v-if="isOidcEnabled" class="google-login-box">
                <a-divider style="border-color: var(--gl-color-line-divider1);color: var(--gl-color-text-level3);font-weight: normal;">
                    {{ $t('login.or') }}
                </a-divider>
                <BaseButton medium class="google-login-btn" :loading="state.oidcLoading" @click="handleLoginWithOidc">
                    <!-- <BaseSvg name="gl-icon-google" :size="20" style="margin-right: 10px;"/> -->
                    {{ $t('login.loginWithOidc') }}
                </BaseButton>
            </div>
        </LoginBox>
    </BaseWhitePage>
</template>

<script setup lang="ts">
import BaseWhitePage from '@/components/base/baseWhitePage.vue'
import LoginBox from './components/loginBox.vue'
import { computed, reactive, ref, onMounted } from 'vue'
import { t } from '@/hooks/useLanguage'
import { useUserStore } from '@/stores/modules/user'
import { useValidateInfo, type FormRules } from 'gl-web-main'
import { BaseRadioButtonsCompact, GlInput, GlPassword } from 'gl-web-main/components'
import { useRouter } from 'vue-router'
import { LoginParams, AuthConfig } from '@/models/user'
import { Form } from 'ant-design-vue'
import { reqAuthConfig } from '@/api/user'
import { useAppStore } from '@/stores/modules/app'
import { useTranslatedOptions } from '@/hooks/useTranslatedOptions'

const AForm = Form
const AFormItem = Form.Item

const router = useRouter()

const { handleValidate } = useValidateInfo<LoginParams>()

const formRef = ref(null)
const authConfig = ref<AuthConfig | null>(null)

// 计算属性来可靠地检查LDAP是否启用 (Computed property to reliably check if LDAP is enabled)
const isLdapEnabled = computed(() => {
    return authConfig.value?.ldapEnabled === true
})

// 计算是否允许OIDC认证
const isOidcEnabled = computed(() => {
    return authConfig.value?.oidcEnabled === true
})

const state = reactive<{formModel: LoginParams, loading: boolean, oidcLoading: boolean}>({
    formModel: {
        username: '',
        password: '',
        authMethod: 'legacy',
    },
    loading: false,
    oidcLoading: false,
})

const LoginTypeOptionsTranslated = computed(() => {
    return useTranslatedOptions([
        { label: t('login.accountLogin'), value: 'legacy' },
        { label: t('login.ldap'), value: 'ldap' },
    ])
})

const formRules = computed<FormRules<LoginParams>>(() => {
    const rules: FormRules<LoginParams> = {
        username: [{ required: true, message: 'login.enterUsernameTip'}],
        password: [{ required: true, message: 'login.enterPwdTip'}],
    }
    return rules
})

// 加载认证配置 (Load authentication configuration)
onMounted(async () => {
    try {
        const response = await reqAuthConfig()
        
        // 提取配置数据 (Extract config data)
        authConfig.value = response.data
        useAppStore().setVersion(authConfig.value.kvmCloudVersion)

        // 若支持LDAP且当前登录方式为legacy，则切换到ldap (If LDAP is supported and current auth method is legacy, switch to ldap)
        if (authConfig.value.ldapEnabled && state.formModel.authMethod === 'legacy') {
            state.formModel.authMethod = 'ldap'
        }
    } catch (error) {
        console.error('Failed to load auth config:', error)
        // 回退 - 无LDAP可用 (Fallback - no LDAP available)
        authConfig.value = { ldapEnabled: false, legacyPassword: true, oidcEnabled: false, kvmCloudVersion: '' }
    }
})

// 登录按钮
const handleLogin = () => {
    formRef.value.validate().then(async () => {
        state.loading = true
        try {
            const loginData: LoginParams = {
                username: state.formModel.username,
                password: state.formModel.password,
                authMethod: state.formModel.authMethod,
            }
            
            await useUserStore().login(loginData)
            // 登录成功后跳转到首页或之前尝试访问的页面
            const redirect = router.currentRoute.value.query.redirect as string || '/' 
            console.log(redirect)
            
            state.loading = false
            router.push(redirect)
        } catch (error) {
            console.log(error)
            state.loading = false
        }
    })
}

// oidc登录按钮
const handleLoginWithOidc = () => {
    state.oidcLoading = true
    const baseUrl = window.location.origin
    window.location.href = `${baseUrl}/auth/oidc/login`
}
</script>

<style scoped lang="scss">
.google-login-box {
    width: 100%;
    
    .google-login-btn {
        width: 100%;
        color: var(--gl-color-text-google);
        background-color: var(--gl-color-bg-google);
        border-color: var(--gl-color-line-google);
        font-size: 14px;
        font-weight: 500;
        height: 40px;
        border-radius: 64px;

        display: flex;
        justify-content: center;
        align-items: center;
    }
}
</style>