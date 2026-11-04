# 用 Docker Compose 部署

本指南介绍如何用本目录的 Docker Compose 文件手动部署 GLKVM 轻量云。一键安装脚本、需要开放的端口和升级方法，见[主 README](../README.zh-CN.md)。

服务器上需要先装好 Docker，以及 1.27 或以上版本的 Docker Compose（或 `docker compose` 插件）。Ubuntu 22.04 及以上、Debian 12 可以用 `apt install -y docker.io docker-compose` 安装，其他系统请参考 [Docker 官方安装文档](https://docs.docker.com/engine/install/)。

## 1. 克隆仓库并准备环境模板
```bash
git clone https://github.com/gl-inet/glkvm-cloud.git
cd glkvm-cloud/docker-compose/
```

* x86_64（amd64）平台：

  ```bash
  cp .env.example .env
  ```
* arm64（AArch64）平台：

  ```bash
  cp .env.arm64.example .env
  ```

两个模板只有镜像（`GLKVM_IMAGE`、`COTURN_IMAGE`）不同。

## 2. 配置环境变量
编辑 `.env` 文件，填写以下必填参数：

- `RTTYS_PASS`：管理员密码（必填，不填平台不启动）
- `RTTYS_TOKEN`：用旧脚本接入的设备在 `RTTYS_DEVICE_PORT` 上使用的令牌（`RTTYS_LEGACY_TOKEN_ACCESS` 为 `true` 时必填；没有这类设备的新安装可以直接关掉旧 token 接入）
- `TURN_USER` / `TURN_PASS`：coturn 鉴权凭据（必填，不填 coturn 不启动）

以上都没有默认值，请自行生成，例如 `openssl rand -hex 16`。如果仍在使用旧版 `.env` 示例里的值，平台启动时会给出警告。

### 管理员账号
- `RTTYS_ADMIN_NAME`：管理员用户名，只能用字母和数字（默认 `admin`），密码就是 `RTTYS_PASS`

### 设备接入
- `GLKVM_ACCESS_IP`：服务器公网地址，用于 TURN；未设置 `DEVICE_ACCESS_HOST` 时也写入接入码（留空则启动时自动检测公网 IP；内网部署请填服务器的内网 IP）
- `RTTYS_DEVICE_MTLS_PORT`：用接入码接入的设备连接的端口（默认 `5915`）
- `DEVICE_ACCESS_HOST`：写入接入码、设备连接该端口用的地址，可以是 IP 或域名（留空则使用 `GLKVM_ACCESS_IP`）。修改后已接入的设备不受影响
- `RTTYS_LEGACY_TOKEN_ACCESS`：是否继续允许用旧脚本接入的设备通过 `RTTYS_DEVICE_PORT` 用 `RTTYS_TOKEN` 连接（默认 `true`）。已弃用：所有设备都改用接入码后请设为 `false`，新安装一开始就可以设为 `false`，这时不需要 `RTTYS_TOKEN` 和 `RTTYS_DEVICE_PORT`
- `SELFHOST_WEBUI_URL`：下发给接入码设备的平台地址，设备本地页面用它显示平台链接，例如 `https://www.example.com`（留空则依次取 `WEB_UI_HOST`、`DEVICE_ACCESS_HOST`、`GLKVM_ACCESS_IP`，再加上 `RTTYS_WEBUI_PORT` 拼出；平台在反向代理后面时请填写）

### 端口
- `RTTYS_WEBUI_PORT`：平台 Web 界面（默认 `443`）
- `RTTYS_HTTP_PROXY_PORT`：设备远程访问（默认 `10443`）
- `RTTYS_DEVICE_PORT`：用旧脚本接入的设备（默认 `5912`）
- `TURN_PORT`：TURN 服务，TCP 和 UDP（默认 `3478`）

请在云安全组和防火墙放行以上端口和 `RTTYS_DEVICE_MTLS_PORT`。使用[反向代理模式](#反向代理模式可选)时，`RTTYS_WEBUI_PORT` 和 `RTTYS_HTTP_PROXY_PORT` 反而不要放行。

### 平台访问域名（可选）
- `WEB_UI_HOST`：只允许通过这个域名访问平台界面和接口，例如 `www.example.com`（留空则不限制）

### LDAP 认证（可选）
- `LDAP_ENABLED`：设为 `true` 启用 LDAP（默认 `false`）
- `LDAP_SERVER`：LDAP 服务器域名或 IP
- `LDAP_PORT`：端口（默认 `389`，TLS 使用 `636`）
- `LDAP_USE_TLS`：设为 `true` 启用 TLS 加密（默认 `false`）
- `LDAP_BIND_DN`：服务账号 DN
- `LDAP_BIND_PASSWORD`：服务账号密码
- `LDAP_BASE_DN`：用户查询的 Base DN
- `LDAP_USER_FILTER`：用户查询过滤器（默认 `(uid=%s)`）
- `LDAP_ALLOWED_GROUPS`：允许访问的群组列表（可选）
- `LDAP_ALLOWED_USERS`：允许访问的用户列表（可选）
- `LDAP_ADMIN_GROUP` / `LDAP_ADMIN_USERS`：给予管理员角色的群组和用户名，逗号分隔（可选）。其他 LDAP 用户为运维员角色

### OIDC 认证（可选）
- `OIDC_ENABLED`：设为 `true` 启用 OIDC（默认 `false`）
- `OIDC_PRIMARY_LOGIN`：设为 `true` 时登录页以 OIDC 为主，账号密码登录收进一个链接（默认 `false`）
- `OIDC_ISSUER`：OIDC Issuer 地址
   示例：`https://accounts.google.com`、`https://your-tenant.auth0.com/`
- `OIDC_CLIENT_ID`：OIDC 客户端 ID
- `OIDC_CLIENT_SECRET`：OIDC 客户端密钥
- `OIDC_AUTH_URL`：授权端点 URL
- `OIDC_TOKEN_URL`：令牌端点 URL
- `OIDC_REDIRECT_URL`：OIDC 回调地址
   域名可自定义，但路径必须为 `/auth/oidc/callback`
   示例：`https://your-domain.example.com/auth/oidc/callback`
- `OIDC_SCOPES`：请求的 OIDC Scope（默认 `"openid profile email"`）
- `OIDC_ALLOWED_USERS`：允许的邮箱或域（可选）
   示例：`user@example.com,@example.com`
- `OIDC_ALLOWED_SUBS`：允许的 OIDC `sub` ID 列表（可选）
- `OIDC_ALLOWED_USERNAMES`：允许的用户名列表（可选）
- `OIDC_ALLOWED_GROUPS`：允许的用户组列表（可选）
- `OIDC_ADMIN_GROUP` / `OIDC_ADMIN_USERS`：给予管理员角色的用户组，以及按 `preferred_username` 或邮箱匹配的用户，逗号分隔（可选）。其他 OIDC 用户为运维员角色

所有配置均在 `.env` 中完成，不需要修改 `docker-compose.yml`、模板或脚本。

## 3. 放置自有证书（可选）
把证书（完整证书链）和私钥放到 `certificate/`，文件名为 `glkvm.cer` 和 `glkvm.key`，用 PEM 格式（Let's Encrypt 的 `fullchain.pem` 和 `privkey.pem`）。没有放证书时，首次启动会自动生成一张证书（浏览器会提示不受信任），见 [`certificate/README.md`](certificate/README.md)。域名和 DNS 的配置见[使用自有域名和 SSL 证书](../README.zh-CN.md#使用自有域名和-ssl-证书可选)。使用反向代理模式时，证书放在 Nginx 上。

## 4. 启动服务
```bash
docker-compose up -d
```

修改 `.env`、模板或证书后，需要重新加载服务：

```bash
docker-compose down && docker-compose up -d
```

使用 Docker CLI 插件时，把 `docker-compose` 换成 `docker compose`。

用 `docker-compose ps` 检查两个服务是否都在运行。如果某个服务一直在重启，用 `docker-compose logs rttys coturn` 查看原因，通常是有必填项没填。

## 5. 登录平台
通过以下地址访问平台，用 `admin`（或 `RTTYS_ADMIN_NAME`）和 `RTTYS_PASS` 登录：

```text
https://<你的服务器公网 IP>
```

通过 IP 访问时，浏览器会提示证书不受信任。

请备份 `database/`（例如 `cp -a database ../database-backup`）：里面有数据库和 `pki`，用接入码接入的设备信任的 CA 就在 `pki` 里。升级方法见[升级](../README.zh-CN.md#升级)。

## 反向代理模式（可选）

GLKVM Cloud 可以运行在 Nginx 等反向代理之后，由反向代理终止 TLS。平台 Web 界面和设备远程访问共用一个 HTTPS 端口（通常为 443），GLKVM Cloud 在后面提供明文 HTTP。

1. **在 `.env` 中设置：**

   ```env
   REVERSE_PROXY_ENABLED=true
   RTTYS_WEBUI_PORT=1443
   SELFHOST_WEBUI_URL=https://www.example.com
   ```

   - `RTTYS_WEBUI_PORT`：把平台 Web 界面从 443 挪开，443 留给 Nginx。`1443` 和 Nginx 示例一致
   - `SELFHOST_WEBUI_URL`：平台对外的访问地址。不填时，设备页面上的平台链接会带上内部端口
   - 设备仍然直连 `RTTYS_DEVICE_MTLS_PORT` 和 `TURN_PORT`，不要放到 Nginx 后面：设备接入端口要自己校验每台设备的出厂证书

2. **不要对外开放明文端口。** `RTTYS_WEBUI_PORT` 和 `RTTYS_HTTP_PROXY_PORT` 是明文 HTTP，Docker 会把它们映射到所有网卡上，并绕过 ufw 等主机防火墙。请不要在云安全组放行这两个端口，只让 Nginx 访问。

3. **配置 Nginx**，参考 [`nginx-reverse-proxy-example.conf`](nginx-reverse-proxy-example.conf)：把 `example.com` 换成你的域名，`/path/to/` 换成证书文件的路径。设备可以通过自己的子域名 `https://<deviceId>.example.com` 打开，需要泛域名解析和泛域名证书；也可以按路径在平台域名下打开，见[按路径访问设备](#按路径访问设备可选)。

   示例里两种方式各有一个平台 Web 界面的 server 块，只能保留其中一个。两个都留着时，Nginx 不会报错，只告警并忽略第二个块，按路径访问就不会生效。

### 必需的反向代理请求头

反向代理必须转发以下请求头，否则可能生成带内部端口（如 `:10443`）的访问地址；还要转发 WebSocket 升级（`$connection_upgrade` 来自 Nginx 示例开头的 `map`）：

```nginx
proxy_set_header Host                $host;
proxy_set_header X-Forwarded-Host    $host;
proxy_set_header X-Forwarded-Proto   $scheme;
proxy_set_header X-Forwarded-Port    $server_port;
proxy_set_header X-Real-IP           $remote_addr;
proxy_set_header X-Forwarded-For     $proxy_add_x_forwarded_for;
proxy_set_header Upgrade             $http_upgrade;
proxy_set_header Connection          $connection_upgrade;
```

### 设备远程访问域名（可选）

```env
DEVICE_ENDPOINT_HOST=
```

- **仅在** `REVERSE_PROXY_ENABLED=true` 时生效
- 用于指定设备远程访问使用的域名
- 生成的设备访问地址格式为：

```text
https://<deviceId>.<DEVICE_ENDPOINT_HOST>/
```

**说明：**

- 不需要包含 `http(s)://` 或路径
- 可与 Web UI 域名不同。设备通过它的子域名打开，所以它需要自己的泛域名解析和证书，例如 `*.kvm.example.com`
- 留空时，将从 `X-Forwarded-*` 请求头自动推导

**示例：**

```text
https://www.example.com             → Web UI
https://<deviceId>.kvm.example.com  → 设备远程访问
DEVICE_ENDPOINT_HOST=kvm.example.com
```

### 按路径访问设备（可选）

设备可以直接挂在 Web UI 的域名下访问，不再每台设备一个子域名。只需要一张证书、一条域名解析，不需要泛域名：

```text
https://www.example.com                     → Web UI
https://www.example.com/access/<deviceId>/  → 设备远程访问
```

除了上面的反向代理模式设置，只需要配置 Nginx：使用 [`nginx-reverse-proxy-example.conf`](nginx-reverse-proxy-example.conf) 里按路径访问的 server 块。它把 `/access/<deviceId>` 转到设备访问端口，并给 Web UI 的请求加上 `X-KVM-Access-Style: path`。

- 只有用接入码接入的设备按路径访问。用旧脚本接入的设备仍然使用自己的子域名 `<deviceId>.<域名>`，需要泛域名配置：有这类设备时请保留 `*.example.com` 的 server 块
- 同一个浏览器可以同时打开多台设备
- 必须在前面配 Nginx，通过 IP 访问平台时不可用
