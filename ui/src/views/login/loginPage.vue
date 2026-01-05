<!--
 * @Author: LPY
 * @Date: 2025-05-30 10:48:43
 * @LastEditors: LPY
 * @LastEditTime: 2026-01-05 14:25:45
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
                
                <!-- 认证选项帮助 - 仅在启用LDAP时显示 (Auth options help - only show when LDAP is enabled) -->
                <div v-if="isLdapEnabled" class="auth-help-container">
                    <div 
                        class="help-icon" 
                        @mouseenter="showTooltip = true" 
                        @mouseleave="showTooltip = false"
                    >
                        <svg 
                            width="16" 
                            height="16" 
                            viewBox="0 0 16 16" 
                            fill="currentColor" 
                            style="margin-left: 8px; color: #656d76; cursor: help;"
                        >
                            <path d="M8 15A7 7 0 1 1 8 1a7 7 0 0 1 0 14zm0 1A8 8 0 1 0 8 0a8 8 0 0 0 0 16z"/>
                            <path d="M5.255 5.786a.237.237 0 0 0 .241.247h.825c.138 0 .248-.113.266-.25.09-.656.54-1.134 1.342-1.134.686 0 1.314.343
                             1.314 1.168 0 .635-.374.927-.965 1.371-.673.489-1.206 1.06-1.168 1.987l.003.217a.25.25 0 0 0 .25.246h.811a.25.25 0 0 0
                              .25-.25v-.105c0-.718.273-.927 1.01-1.486.609-.463 1.244-.977 1.244-2.056 0-1.511-1.276-2.241-2.673-2.241-1.267 0-2.655
                              .59-2.75 2.286zm1.557 5.763c0 .533.425.927 1.01.927.609 0 1.028-.394 1.028-.927 0-.552-.42-.94-1.029-.94-.584 0-1.009
                              .388-1.009.40z"/>
                        </svg>
                    </div>
                    <!-- 提示框 (Tooltip) -->
                    <div v-show="showTooltip" class="auth-tooltip">
                        <div style="font-weight: bold; margin-bottom: 4px;">{{ $t('login.authOptions') }}:</div>
                        <div>• {{ $t('login.ldapAuth') }}</div>
                        <div>• {{ $t('login.webManagementAuth') }}</div>
                    </div>
                </div>
            </div>

            <AForm 
                class="dense-form" 
                ref="formRef"
                :model="state.formModel"
                :rules="formRules"
                :validateTrigger="['blur', 'change']"
                style="width: 100%;"
                @validate="handleValidate"
            >
                <!-- 用户名字段 - 仅在启用LDAP时显示 (Username field - only show if LDAP is enabled) -->
                <AFormItem v-if="isLdapEnabled" name="username">
                    <AInput
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
import { GlPassword } from 'gl-web-main/components'
import { useRouter } from 'vue-router'
import { LoginParams, AuthConfig } from '@/models/user'
import { message, Input, Form } from 'ant-design-vue'
import { reqAuthConfig } from '@/api/user'
import { useAppStore } from '@/stores/modules/app'

const AInput = Input
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
    },
    loading: false,
    oidcLoading: false,
})

const showTooltip = ref(false)

const formRules = computed<FormRules<LoginParams>>(() => {
    const rules: FormRules<LoginParams> = {
        password: [{ required: true, message: 'login.enterPwdTip'}],
    }
    
    // 用户名为可选字段，支持双重认证模式 (Username is optional, supporting dual authentication modes)
    return rules
})

// 加载认证配置 (Load authentication configuration)
onMounted(async () => {
    try {
        const response = await reqAuthConfig()
        
        // 提取配置数据 (Extract config data)
        const configData = response?.info || response?.data?.info || response?.data || response
        authConfig.value = configData
        useAppStore().setVersion(configData.kvmCloudVersion)
    } catch (error) {
        console.error('Failed to load auth config:', error)
        // 回退 - 无LDAP可用 (Fallback - no LDAP available)
        authConfig.value = { ldapEnabled: false, legacyPassword: true, oidcEnabled: false }
    }
})

// 登录按钮
const handleLogin = () => {
    formRef.value.validate().then(async () => {
        state.loading = true
        try {
            // 基于用户名自动确定认证方法 (Auto-determine auth method based on username)
            const loginData: LoginParams = {
                username: state.formModel.username,
                password: state.formModel.password,
                authMethod: (state.formModel.username && authConfig.value?.ldapEnabled) ? 'ldap' : 'legacy',
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
            
            // 检查后端返回的错误类型 (Check error type returned by backend)
            const errorData = error?.response?.data
            const backendError = errorData?.error || ''
            
            // 根据后端返回的具体错误类型显示不同消息 (Show different messages based on specific error type from backend)
            if (backendError === 'user not authorized') {
                // 授权失败 - 用户存在但权限不足 (Authorization failure - user exists but insufficient permissions)
                message.error(t('login.notAuthorized'))
            } else {
                // 认证失败 - 用户名/密码错误 (Authentication failure - incorrect username/password)
                message.error(t('login.incorrectPwd'))
            }
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
.auth-help-container {
  position: relative;
  display: inline-block;
}

.help-icon {
  display: inline-flex;
  align-items: center;
  transition: color 0.2s ease;
  
  &:hover {
    color: #0066cc !important;
  }
}

.auth-tooltip {
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  margin-top: 8px;
  padding: 12px;
  background-color: #2c3e50;
  color: white;
  border-radius: 6px;
  font-size: 14px;
  width: 320px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  
  // 添加箭头 (Add arrow)
  &::before {
    content: '';
    position: absolute;
    top: -6px;
    left: 50%;
    transform: translateX(-50%);
    width: 0;
    height: 0;
    border-left: 6px solid transparent;
    border-right: 6px solid transparent;
    border-bottom: 6px solid #2c3e50;
  }
  
  // 确保多行文本正确显示 (Ensure multi-line text displays correctly)
  div {
    white-space: normal;
    line-height: 1.4;
    
    &:not(:last-child) {
      margin-bottom: 4px;
    }
  }
}

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