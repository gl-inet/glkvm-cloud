/**
 * Notification Settings API
 */

import request from './request'

// ─── SMTP Config ────────────────────────────────────────────────

export interface SMTPConfig {
    host: string
    port: number
    username: string
    password: string
    fromEmail: string
    encryption: string
    enabled: boolean
    updatedAt: number
}

export function reqGetSMTPConfig () {
    return request<SMTPConfig>({ url: '/api/notification/smtp' })
}

export function reqSaveSMTPConfig (data: Omit<SMTPConfig, 'updatedAt'>) {
    return request<SMTPConfig>({ url: '/api/notification/smtp', method: 'PUT', data })
}

export function reqTestSMTP (email: string) {
    return request<{ message: string }>({ url: '/api/notification/smtp/test', method: 'POST', data: { email } })
}

// ─── Notify Rules ───────────────────────────────────────────────

export interface NotifyRules {
    deviceOnline: boolean
    deviceOffline: boolean
    remoteAccess: boolean
    updatedAt: number
}

export function reqGetNotifyRules () {
    return request<NotifyRules>({ url: '/api/notification/rules' })
}

export function reqSaveNotifyRules (data: Omit<NotifyRules, 'updatedAt'>) {
    return request<NotifyRules>({ url: '/api/notification/rules', method: 'PUT', data })
}

// ─── Recipients ─────────────────────────────────────────────────

export interface Recipient {
    id: number
    email: string
    createdAt: number
}

export interface ListRecipientsResp {
    items: Recipient[]
}

export function reqListRecipients () {
    return request<ListRecipientsResp>({ url: '/api/notification/recipients' })
}

export function reqAddRecipient (email: string) {
    return request<Recipient>({ url: '/api/notification/recipients', method: 'POST', data: { email } })
}

export function reqRemoveRecipient (id: number) {
    return request<{ message: string }>({ url: `/api/notification/recipients/${id}`, method: 'DELETE' })
}
