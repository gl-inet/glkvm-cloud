/*
 * @Author: LPY
 * @Date: 2025-06-03 10:52:47
 * @LastEditors: LPY
 * @LastEditTime: 2026-02-04 12:10:47
 * @FilePath: \glkvm-cloud\ui\src\api\requestError.ts
 * @Description: 统一错误提示。
 */

import { t } from '@/hooks/useLanguage'
import { RequestErrorCodeMap, type RequestErrorCodeEnum } from '@/models/request'
import { message } from 'ant-design-vue'

// 错误码翻译的统一前缀
export const ERR_PREFIX = 'errorCode'

const reflectionCode = function (response: { code: RequestErrorCodeEnum, message: string }) {
    const { code, message } = response
    if (RequestErrorCodeMap.get(code)) {
        return t(`${ERR_PREFIX}.${RequestErrorCodeMap.get(code)}`)
    } else {
        /** 防止报一长串的后端错误信息 */
        const errString = 'Server Error Message'
        const msgString = typeof message === 'string' ? message : errString
        return (msgString.length > 100) ? errString : (msgString || errString)
    }
}

/** 防止一次性显示多条错误信息 */
let preventDuplicate = false
const showErrorMessage = function (response: { code: RequestErrorCodeEnum, message: string }, data = { translate: true }) {
    if (!preventDuplicate) {
        preventDuplicate = true
        const { translate } = data
        message.error(translate ? reflectionCode(response) : response.message)
        setTimeout(function () {
            preventDuplicate = false
        }, 3000)
    }
}

export {
    showErrorMessage,
}