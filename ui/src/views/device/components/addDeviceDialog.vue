<!--
 * @Author: LPY
 * @Date: 2025-08-25 09:32:42
 * @LastEditors: LPY
 * @LastEditTime: 2026-03-25 10:11:49
 * @FilePath: \glkvm-cloud\ui\src\views\device\components\addDeviceDialog.vue
 * @Description: 添加设备弹窗
-->
<template>
    <BaseModal
        :width="560"
        :open="props.open"
        :title="$t('device.addDevice')"
        destroyOnClose
        :showCancel="false"
        :okText="state.mode === AddDeviceModeEnum.ACCESS_CODE ? $t('device.copyAccessCode') : $t('device.copyScript')"
        @ok="handleCopy"
        @close="emits('update:open', false)"
    >
        <div class="top-tips">
            <div class="tips-icon">
                <BaseSvg name="gl-icon-device" :size="16"></BaseSvg>
            </div>
            <BaseText>{{ state.mode === AddDeviceModeEnum.ACCESS_CODE ? $t('device.accessCodeTip') : $t('device.addDeviceTip') }}</BaseText>
        </div>
        <!-- 切换添加方式 -->
        <BaseRadioButtonsCompact
            v-model:value="state.mode"
            :options="modeOptions.value"
            style="width: 100%; margin: 12px 0;"
        />

        <template v-if="state.mode === AddDeviceModeEnum.ACCESS_CODE">
            <div class="generate-row">
                <BaseText>{{ $t('device.accessCodeCount') }}</BaseText>
                <AInputNumber v-model:value="state.count" :min="1" :max="MAX_CODES" :precision="0" />
                <BaseButton type="primary" :loading="state.generating" @click="handleGenerate">
                    {{ $t('device.generateAccessCode') }}
                </BaseButton>
            </div>
            <template v-if="state.generated.length">
                <BaseInfo warning style="font-size: 12px;">{{ $t('device.accessCodeShownOnce') }}</BaseInfo>
                <pre class="script-box code-box"><BaseText type="body-r">{{ generatedText }}</BaseText></pre>
            </template>

            <BaseText type="body-m" class="codes-title">{{ $t('device.recentAccessCodes') }}</BaseText>
            <div class="codes-list">
                <BaseText v-if="!state.codes.length" type="caption-r" class="codes-empty">{{ $t('device.noAccessCodes') }}</BaseText>
                <div v-for="item in state.codes" :key="item.id" class="code-row">
                    <BaseText type="caption-r" class="code-time">{{ formatTime(item.createdAt) }}</BaseText>
                    <BaseTag :primary="item.status === 'unused'">{{ $t(`device.accessCodeStatus.${item.status}`) }}</BaseTag>
                    <BaseText type="caption-r" class="code-device">{{ item.devid || '' }}</BaseText>
                    <APopconfirm
                        v-if="item.status === 'unused'"
                        :title="$t('device.revokeAccessCodeConfirm')"
                        @confirm="handleRevoke(item.id)"
                    >
                        <BaseButton danger size="small">{{ $t('device.revokeAccessCode') }}</BaseButton>
                    </APopconfirm>
                </div>
            </div>
        </template>

        <pre v-else class="script-box">
            <BaseText type="body-r">{{ state.scriptContent }}</BaseText>
        </pre>
    </BaseModal>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { BaseInfo, BaseRadioButtonsCompact } from 'gl-web-main/components'
import BaseModal from '@/components/base/baseModalI18n.vue'
import { createRegistrationCodesApi, getAddDeviceScriptInfoApi, listRegistrationCodesApi, revokeRegistrationCodeApi } from '@/api/device'
import { copyText } from 'gl-web-main'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'
import { t } from '@/hooks/useLanguage'
import { AddDeviceModeEnum, addDeviceModeLabelMap } from '@/models/setting'
import type { CreatedRegistrationCode, RegistrationCode } from '@/models/device'
import { useTranslatedOptions } from '@/hooks/useTranslatedOptions'

/** 一次最多生成的接入码数（与后端一致） */
const MAX_CODES = 20

const props = defineProps<{ open: boolean }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
}>()

const state = reactive({
    mode: AddDeviceModeEnum.ACCESS_CODE,
    count: 1,
    generating: false,
    generated: [] as CreatedRegistrationCode[],
    codes: [] as RegistrationCode[],
    scriptData: {},
    scriptContent: '',
})

const modeOptions = computed(() => {
    return useTranslatedOptions([
        { label: addDeviceModeLabelMap.get(AddDeviceModeEnum.ACCESS_CODE), value: AddDeviceModeEnum.ACCESS_CODE },
        { label: addDeviceModeLabelMap.get(AddDeviceModeEnum.LEGACY_SCRIPT), value: AddDeviceModeEnum.LEGACY_SCRIPT },
    ])
})

const generatedText = computed(() => state.generated.map(item => item.accessCode).join('\n'))

const formatTime = (ts: number) => dayjs.unix(ts).format('YYYY-MM-DD HH:mm')

/** 旧固件：GL-iNet KVM 自部署云脚本（token 接入） */
const generateScript = (
    hostname: string, port: string, token: string,
    webrtcIP: string, webrtcPort: string, webrtcUsername: string, webrtcPassword: string,
    webUIURL: string,
) => {
    return `#!/bin/sh

HOSTNAME="${hostname}"
PORT="${port}"
TOKEN="${token}"
WEBRTC_IP="${webrtcIP}"
WEBRTC_PORT="${webrtcPort}"
WEBRTC_USERNAME="${webrtcUsername}"
WEBRTC_PASSWORD="${webrtcPassword}"
WEBUI_URL="${webUIURL}"

TARGET_DIR="/etc/kvmd/user/scripts"
SCRIPT_FILE="$TARGET_DIR/S01selfCloud"
WATCHDOG_SCRIPT="$TARGET_DIR/rtty-loop.sh"
CLOUD_CONFIG="/etc/kvmd/user/selfhost-cloud.json"

# 1. Create directory
mkdir -p "$TARGET_DIR"

# 2. Write selfhost cloud config JSON
cat <<CLOUDCFG > "$CLOUD_CONFIG"
{
    "webui_url": "$WEBUI_URL",
    "updated_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
CLOUDCFG

# 3. Write S01selfCloud script
cat <<'EOF' > "$SCRIPT_FILE"
#!/bin/sh

HOSTNAME="${hostname}"
PORT="${port}"
TOKEN="${token}"
WEBRTC_IP="${webrtcIP}"
WEBRTC_PORT="${webrtcPort}"
WEBRTC_USERNAME="${webrtcUsername}"
WEBRTC_PASSWORD="${webrtcPassword}"

TARGET_DIR="/etc/kvmd/user/scripts"
SCRIPT_FILE="$TARGET_DIR/S01selfCloud"
WATCHDOG_SCRIPT="$TARGET_DIR/rtty-loop.sh"

start() {
    if pgrep -f "$WATCHDOG_SCRIPT" > /dev/null; then
        echo "selfHosted cloud is already started"
        return 0
    fi

    echo "starting self hosted cloud"

    # Write TURN config
    cat <<TURNCONF > /tmp/turnserver.json
{
    "username": "$WEBRTC_USERNAME",
    "ttl": 864000,
    "password": "$WEBRTC_PASSWORD",
    "uris": [
        "turn:$WEBRTC_IP:$WEBRTC_PORT?transport=udp",
        "turn:$WEBRTC_IP:$WEBRTC_PORT?transport=tcp"
    ]
}
TURNCONF

    device_id=$(cat /proc/gl-hw-info/device_ddns)
    device_mac=$(cat /proc/gl-hw-info/device_mac)

    mkdir -p $(dirname "$WATCHDOG_SCRIPT")

    cat <<RTTY > "$WATCHDOG_SCRIPT"
#!/bin/sh
device_id=$(cat /proc/gl-hw-info/device_ddns)
device_mac=$(cat /proc/gl-hw-info/device_mac)
while true; do
    if ! pgrep -f "rtty.*-d $device_mac" > /dev/null; then
        echo "rtty not running, starting..."
        rtty -sx -T 2 -I "$device_id" -h $HOSTNAME$PORT -t "$TOKEN" -d "$device_mac" &
    fi
    sleep 5
done
RTTY

    chmod +x "$WATCHDOG_SCRIPT"
    nohup "$WATCHDOG_SCRIPT" >> /tmp/rtty.log 2>&1 &
    echo "started selfHosted cloud"
}

stop() {
    echo "stopping selfHosted cloud"
    device_mac=$(cat /proc/gl-hw-info/device_mac)
    pkill -f "$WATCHDOG_SCRIPT"
    pkill -f "rtty.*-d $device_mac"
    echo "stopped selfHosted cloud"
}

restart() {
    stop
    sleep 1
    start
}

case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart|reload)
        restart
        ;;
    *)
        echo "Usage: $0 {start|stop|restart}"
        exit 1
esac

exit 0
EOF

# 5. Add execution permissions
chmod +x "$SCRIPT_FILE"

# 6. Execute restart once
"$SCRIPT_FILE" restart

`
}

/** 组装脚本 */
const handleGenerateScript = () => {
    const { hostname, port, token, webrtcIP, webrtcPort, webrtcUsername, webrtcPassword, webUIURL } = state.scriptData as any
    state.scriptContent = generateScript(hostname, port, token, webrtcIP, webrtcPort, webrtcUsername, webrtcPassword, webUIURL)
}

/** 生成接入码 */
const handleGenerate = async () => {
    state.generating = true
    try {
        const res = await createRegistrationCodesApi(state.count)
        state.generated = res.data.items
        await loadCodes()
    } finally {
        state.generating = false
    }
}

/** 作废接入码 */
const handleRevoke = async (id: number) => {
    await revokeRegistrationCodeApi(id)
    message.success(t('device.revokeAccessCodeSuccess'))
    await loadCodes()
}

/** 复制：接入码页复制新生成的接入码，脚本页复制脚本 */
const handleCopy = () => {
    const text = state.mode === AddDeviceModeEnum.ACCESS_CODE ? generatedText.value : state.scriptContent
    if (!text) {
        message.warning(t('device.generateAccessCodeFirst'))
        return
    }
    try {
        copyText(text)
        message.success(t('common.copySuccess'))
    } catch {
        message.error(t('common.copyFailed'))
    }
}

const loadCodes = async () => {
    const res = await listRegistrationCodesApi()
    state.codes = res.data.items
}

/** 初始化数据 */
watch(() => props.open, (newVal) => {
    if (newVal) {
        init()
    }
})

const init = async () => {
    state.generated = []
    state.count = 1
    loadCodes()
    const res = await getAddDeviceScriptInfoApi()
    state.scriptData = res.data
    handleGenerateScript()
}
</script>

<style lang="scss" scoped>
.script-box {
  border-radius: 6px;
  background: var(--gl-color-bg-surface2);
  padding: 10px;
  overflow: auto;
  max-height: 375px;
  font-family: unset;
}

.top-tips {
  display: flex;
  align-items: center;
  padding: 8px 10px;
  border: 1px solid var(--gl-color-brand-primary);
  background-color: var(--gl-color-brand-background);
  border-radius: 6px;

  .tips-icon {
    display: flex;
    align-items: center;
    margin-right: 10px;
    color: var(--gl-color-brand-primary);
  }
}

.generate-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.code-box {
  margin: 8px 0 0;
  max-height: 160px;
  white-space: pre-wrap;
  word-break: break-all;
}

.codes-title {
  display: block;
  margin: 16px 0 8px;
}

.codes-list {
  max-height: 220px;
  overflow: auto;
}

.codes-empty {
  color: var(--gl-color-text-secondary);
}

.code-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 32px;
  border-bottom: 1px solid var(--gl-color-bg-surface2);

  .code-time {
    width: 120px;
    flex-shrink: 0;
  }

  .code-device {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>