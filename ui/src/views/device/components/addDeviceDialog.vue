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
        :width="500"
        :open="props.open"
        :title="$t('device.addDevice')"
        destroyOnClose
        :showCancel="false"
        :okText="$t('device.copyScript')"
        @ok="handleCopy"
        @close="emits('update:open', false)"
    >
        <div class="top-tips">
            <div class="tips-icon">
                <BaseSvg name="gl-icon-device" :size="16"></BaseSvg>
            </div>
            <BaseText>{{ $t('device.addDeviceTip') }}</BaseText>
        </div>
        <!-- 切换添加方式 -->
        <BaseRadioButtonsCompact
            v-model:value="state.operatingSystem"
            :options="OperatingSystemTranslated.value"
            @update:value="handleGenerateScript"
            style="width: 100%; margin: 12px 0;"
        />
        <BaseInfo v-if="state.operatingSystem === OperatingSystemEnum.LINUX" warning style="font-size: 12px;">
            {{ $t('device.linuxTips') }}
        </BaseInfo>
        <pre class="script-box">
            <BaseText type="body-r">{{ state.scriptContent }}</BaseText>
        </pre>
    </BaseModal>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { BaseInfo, BaseModal, BaseRadioButtonsCompact } from 'gl-web-main/components'
import { getAddDeviceScriptInfoApi } from '@/api/device'
import { copyText } from 'gl-web-main'
import { message } from 'ant-design-vue'
import { t } from '@/hooks/useLanguage'
import { OperatingSystemEnum, operatingSystemLabelMap } from '@/models/setting'
import { useTranslatedOptions } from '@/hooks/useTranslatedOptions'

const props = defineProps<{ open: boolean }>()

const emits = defineEmits<{
    (e: 'update:open', value: boolean): void;
}>()

const state = reactive({
    scriptData: {},
    scriptContent: '',
    operatingSystem: OperatingSystemEnum.GL_KVM,
})

const OperatingSystemTranslated = computed(() => {
    return useTranslatedOptions([
        { label: operatingSystemLabelMap.get(OperatingSystemEnum.GL_KVM), value: OperatingSystemEnum.GL_KVM },
        { label: operatingSystemLabelMap.get(OperatingSystemEnum.LINUX), value: OperatingSystemEnum.LINUX },
        // { label: operatingSystemLabelMap.get(OperatingSystemEnum.WINDOWS), value: OperatingSystemEnum.WINDOWS },
        // { label: operatingSystemLabelMap.get(OperatingSystemEnum.MAC_OS), value: OperatingSystemEnum.MAC_OS },
    ])
})

const generateScript = (hostname: string, port:string, token: string, webrtcIP: string, webrtcPort: string, webrtcUsername: string, webrtcPassword: string) => {
    if (state.operatingSystem === OperatingSystemEnum.GL_KVM) {
        return `#!/bin/sh

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

# 1. Create directory
mkdir -p "$TARGET_DIR"

# 2. Write S01selfCloud script
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
        rtty -sx -T 2 -I "$device_id" -h $HOSTNAME$PORT -t $TOKEN -d "$device_mac" &
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

# 3. Add execution permissions
chmod +x "$SCRIPT_FILE"

# 4. Execute restart once
"$SCRIPT_FILE" restart

`
    } else if (state.operatingSystem === OperatingSystemEnum.MAC_OS) {
        return `curl -fsSL https://kvm-cloud.gl-inet.com/selfhost/clients/install-macos.sh | sudo bash -s -- \\
-h ${hostname} -p ${port} -t ${token}`
    } else if (state.operatingSystem === OperatingSystemEnum.LINUX) {
        return `curl -fsSL https://kvm-cloud.gl-inet.com/selfhost/clients/install-linux.sh | sh -s -- \\
-h ${hostname} -p ${port} -t ${token}`
    } else if (state.operatingSystem === OperatingSystemEnum.WINDOWS) {
        return `powershell -ExecutionPolicy Bypass -Command "iwr https://kvm-cloud.gl-inet.com/selfhost/clients/install-windows.ps1 -OutFile install.ps1;
.\\install.ps1 -Host_Addr ${hostname} -Port ${port} -Token ${token}"`
    }
}

/** 组装脚本 */
const handleGenerateScript = () => {
    const { hostname, port, token, webrtcIP, webrtcPort, webrtcUsername, webrtcPassword } = state.scriptData as any
    state.scriptContent = generateScript(hostname, port, token, webrtcIP, webrtcPort, webrtcUsername, webrtcPassword)
}

/** 复制脚本 */
const handleCopy = () => {
    try {
        copyText(state.scriptContent)
        message.success(t('common.copySuccess'))
    } catch {
        message.error(t('common.copyFailed'))
    }
}

/** 初始化数据 */
watch(() => props.open, (newVal) => {
    if (newVal) {
        init()
    }
})

const init = async () => {
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
</style>