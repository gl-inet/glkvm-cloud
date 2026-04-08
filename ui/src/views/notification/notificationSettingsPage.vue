<!--
 * @Description: Notification Settings Page — SMTP config, rules & recipients
-->
<template>
    <div class="notification-settings">
        <BaseLoadingContainer :spinning="state.loading">
            <div class="notification-settings-content">
                <!-- ═══ SMTP Configuration ═══ -->
                <div class="ns-card">
                    <div class="ns-card-header">
                        <BaseText type="body-b">{{ $t('notification.smtpConfig') }}</BaseText>
                    </div>
                    <div class="ns-card-body">
                        <a-form layout="vertical" :model="state.smtp">
                            <div class="form-row">
                                <a-form-item :label="$t('notification.smtpHost')" class="form-item-half">
                                    <a-input v-model:value="state.smtp.host" placeholder="smtp.gmail.com" />
                                </a-form-item>
                                <a-form-item :label="$t('notification.smtpPort')" class="form-item-quarter">
                                    <a-input-number v-model:value="state.smtp.port" :min="1" :max="65535" style="width: 100%;" />
                                </a-form-item>
                                <a-form-item :label="$t('notification.encryption')" class="form-item-quarter">
                                    <a-select v-model:value="state.smtp.encryption">
                                        <a-select-option value="starttls">STARTTLS</a-select-option>
                                        <a-select-option value="tls">SSL/TLS</a-select-option>
                                        <a-select-option value="none">None</a-select-option>
                                    </a-select>
                                </a-form-item>
                            </div>
                            <div class="form-row">
                                <a-form-item :label="$t('notification.smtpUsername')" class="form-item-half">
                                    <a-input v-model:value="state.smtp.username" placeholder="user@gmail.com" />
                                </a-form-item>
                                <a-form-item :label="$t('notification.smtpPassword')" class="form-item-half">
                                    <a-input-password v-model:value="state.smtp.password" />
                                </a-form-item>
                            </div>
                            <div class="form-row">
                                <a-form-item :label="$t('notification.fromEmail')" class="form-item-half">
                                    <a-input v-model:value="state.smtp.fromEmail" placeholder="noreply@example.com" />
                                </a-form-item>
                                <a-form-item :label="$t('notification.enableNotification')" class="form-item-half" style="padding-top: 8px;">
                                    <a-switch v-model:checked="state.smtp.enabled" />
                                </a-form-item>
                            </div>
                            <div class="form-actions">
                                <BaseButton type="primary" :loading="state.savingSMTP" @click="handleSaveSMTP">
                                    {{ $t('common.apply') }}
                                </BaseButton>
                                <BaseButton style="margin-left: 8px;" :loading="state.testing" @click="handleTestSMTP">
                                    {{ $t('notification.testEmail') }}
                                </BaseButton>
                            </div>
                        </a-form>
                    </div>
                </div>

                <!-- ═══ Notification Rules ═══ -->
                <div class="ns-card">
                    <div class="ns-card-header">
                        <BaseText type="body-b">{{ $t('notification.notifyRules') }}</BaseText>
                    </div>
                    <div class="ns-card-body">
                        <div class="rule-row">
                            <div class="rule-info">
                                <BaseText type="body-r">{{ $t('notification.ruleDeviceOnline') }}</BaseText>
                                <BaseText type="caption" variant="level3">{{ $t('notification.ruleDeviceOnlineDesc') }}</BaseText>
                            </div>
                            <a-switch v-model:checked="state.rules.deviceOnline" @change="handleSaveRules" />
                        </div>
                        <div class="rule-row">
                            <div class="rule-info">
                                <BaseText type="body-r">{{ $t('notification.ruleDeviceOffline') }}</BaseText>
                                <BaseText type="caption" variant="level3">{{ $t('notification.ruleDeviceOfflineDesc') }}</BaseText>
                            </div>
                            <a-switch v-model:checked="state.rules.deviceOffline" @change="handleSaveRules" />
                        </div>
                        <div class="rule-row">
                            <div class="rule-info">
                                <BaseText type="body-r">{{ $t('notification.ruleRemoteAccess') }}</BaseText>
                                <BaseText type="caption" variant="level3">{{ $t('notification.ruleRemoteAccessDesc') }}</BaseText>
                            </div>
                            <a-switch v-model:checked="state.rules.remoteAccess" @change="handleSaveRules" />
                        </div>
                    </div>
                </div>

                <!-- ═══ Recipient Emails ═══ -->
                <div class="ns-card">
                    <div class="ns-card-header">
                        <BaseText type="body-b">{{ $t('notification.recipients') }}</BaseText>
                    </div>
                    <div class="ns-card-body">
                        <div class="recipient-add">
                            <a-input
                                v-model:value="state.newEmail"
                                :placeholder="$t('notification.recipientPlaceholder')"
                                style="width: 320px; margin-right: 8px;"
                                @pressEnter="handleAddRecipient"
                            />
                            <BaseButton type="primary" :loading="state.addingRecipient" @click="handleAddRecipient">
                                {{ $t('notification.addRecipient') }}
                            </BaseButton>
                        </div>
                        <div v-if="state.recipients.length > 0" class="recipient-list">
                            <div v-for="r in state.recipients" :key="r.id" class="recipient-item">
                                <BaseText type="body-r">{{ r.email }}</BaseText>
                                <a-popconfirm
                                    :title="$t('notification.removeRecipientConfirm')"
                                    :ok-text="$t('common.ok')"
                                    :cancel-text="$t('common.cancel')"
                                    @confirm="handleRemoveRecipient(r.id)"
                                >
                                    <BaseButton type="text" danger size="small">{{ $t('common.remove') }}</BaseButton>
                                </a-popconfirm>
                            </div>
                        </div>
                        <a-empty v-else :description="$t('notification.noRecipients')" :image="null" />
                    </div>
                </div>
            </div>
        </BaseLoadingContainer>
    </div>
</template>

<script setup lang="ts">
import { reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { useI18n } from 'vue-i18n'
import {
    reqGetSMTPConfig, reqSaveSMTPConfig, reqTestSMTP,
    reqGetNotifyRules, reqSaveNotifyRules,
    reqListRecipients, reqAddRecipient, reqRemoveRecipient,
    type SMTPConfig, type Recipient,
} from '@/api/notification'

const { t } = useI18n()

const state = reactive({
    loading: false,
    savingSMTP: false,
    testing: false,
    addingRecipient: false,
    smtp: {
        host: '',
        port: 587,
        username: '',
        password: '',
        fromEmail: '',
        encryption: 'starttls',
        enabled: false,
    } as Omit<SMTPConfig, 'updatedAt'>,
    rules: {
        deviceOnline: false,
        deviceOffline: false,
        remoteAccess: false,
    },
    recipients: [] as Recipient[],
    newEmail: '',
})

// ─── Load all data ──────────────────────────────────────────────

const loadAll = async () => {
    state.loading = true
    try {
        const [smtpRes, rulesRes, recipientsRes] = await Promise.all([
            reqGetSMTPConfig(),
            reqGetNotifyRules(),
            reqListRecipients(),
        ])
        if (smtpRes.data) {
            const d = smtpRes.data
            state.smtp = { host: d.host, port: d.port, username: d.username, password: d.password, fromEmail: d.fromEmail, encryption: d.encryption, enabled: d.enabled }
        }
        if (rulesRes.data) {
            state.rules = { deviceOnline: rulesRes.data.deviceOnline, deviceOffline: rulesRes.data.deviceOffline, remoteAccess: rulesRes.data.remoteAccess }
        }
        if (recipientsRes.data) {
            state.recipients = recipientsRes.data.items || []
        }
    } finally {
        state.loading = false
    }
}

// ─── SMTP ───────────────────────────────────────────────────────

const handleSaveSMTP = async () => {
    state.savingSMTP = true
    try {
        await reqSaveSMTPConfig(state.smtp)
        message.success(t('common.success'))
    } catch {
        message.error(t('common.failed'))
    } finally {
        state.savingSMTP = false
    }
}

const handleTestSMTP = async () => {
    if (state.recipients.length === 0 && !state.newEmail) {
        message.warning(t('notification.noRecipientForTest'))
        return
    }
    const email = state.recipients.length > 0 ? state.recipients[0].email : state.newEmail
    state.testing = true
    try {
        await reqTestSMTP(email)
        message.success(t('notification.testEmailSent') + ': ' + email)
    } catch (e: any) {
        message.error(e?.response?.data?.message || t('common.failed'))
    } finally {
        state.testing = false
    }
}

// ─── Rules ──────────────────────────────────────────────────────

const handleSaveRules = async () => {
    try {
        await reqSaveNotifyRules(state.rules)
    } catch {
        message.error(t('common.failed'))
    }
}

// ─── Recipients ─────────────────────────────────────────────────

const handleAddRecipient = async () => {
    const email = state.newEmail.trim()
    if (!email) return
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
        message.warning(t('notification.invalidEmail'))
        return
    }
    state.addingRecipient = true
    try {
        const res = await reqAddRecipient(email)
        if (res.data) {
            state.recipients.push(res.data)
            state.newEmail = ''
        }
    } catch {
        message.error(t('common.failed'))
    } finally {
        state.addingRecipient = false
    }
}

const handleRemoveRecipient = async (id: number) => {
    try {
        await reqRemoveRecipient(id)
        state.recipients = state.recipients.filter(r => r.id !== id)
    } catch {
        message.error(t('common.failed'))
    }
}

onMounted(() => {
    loadAll()
})
</script>

<style scoped lang="scss">
.notification-settings {
    height: 100%;
    overflow-y: auto;
    padding: 0 4px;

    .notification-settings-content {
        display: flex;
        flex-direction: column;
        gap: 16px;
        padding-bottom: 20px;
    }
}

.ns-card {
    background-color: var(--gl-color-bg-surface1);
    border-radius: 10px;
    padding: 20px 24px;

    .ns-card-header {
        margin-bottom: 16px;
        padding-bottom: 12px;
        border-bottom: 1px solid var(--gl-color-line-divider2);
    }

    .ns-card-body {
        .form-row {
            display: flex;
            gap: 16px;
        }
        .form-item-half {
            flex: 1;
        }
        .form-item-quarter {
            flex: 0 0 160px;
        }
        .form-actions {
            margin-top: 8px;
        }
    }
}

.rule-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 0;
    border-bottom: 1px solid var(--gl-color-line-divider2);

    &:last-child {
        border-bottom: none;
    }

    .rule-info {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }
}

.recipient-add {
    display: flex;
    align-items: center;
    margin-bottom: 16px;
}

.recipient-list {
    .recipient-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 8px 12px;
        border-radius: 6px;
        border: 1px solid var(--gl-color-line-divider2);
        margin-bottom: 8px;

        &:hover {
            background-color: var(--gl-color-bg-icon-button-hover);
        }
    }
}
</style>
