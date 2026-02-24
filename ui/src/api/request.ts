/*
 * @Author: LPY
 * @Date: 2025-06-03 09:29:03
 * @LastEditors: LPY
 * @LastEditTime: 2026-01-29 11:15:24
 * @FilePath: \glkvm-cloud\ui\src\api\request.ts
 * @Description: 请求统一配置文件
 */
import { NotNeedHandledRequestErrorCodeList, RequestErrorCodeEnum } from '@/models/request'
import { showErrorMessage } from './requestError'
import { HttpService } from 'gl-web-main'
import type { AxiosResponse } from 'axios'
import { useUserStore } from '@/stores/modules/user'

export const httpService = new HttpService(
    {timeout: 30000},
    config => {
        return config
    },
    response => {
        const code = response.data.code
        if (code === RequestErrorCodeEnum.AUTH_EXPIRED) {
            useUserStore().autoLogout()
            return Promise.reject(response.data)
        }

        if (code !== RequestErrorCodeEnum.SUCCESS) {
            if (!NotNeedHandledRequestErrorCodeList.includes(code)) {
                showErrorMessage(response.data)
            }
            return Promise.reject(response.data)
        }
        
        return response as AxiosResponse
    },
    error => {
        console.log('请求错误', error)
        return Promise.reject(error)
    },
)

export const { httpApiPrefixCloudBasic } = httpService
export default httpService.request