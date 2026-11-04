# 快速开始（Quick Start）

本指南展示如何使用提供的 Docker Compose 环境模板部署 **glkvm-cloud**。

### 1. **克隆仓库并准备环境模板**

```bash
git clone https://github.com/gl-inet/glkvm-cloud.git
cd glkvm-cloud/docker-compose/
```

* **x86_64（amd64）平台**：
  ```bash
  cp .env.example .env
  ```
* **arm64（AArch64）平台**：
  
  ```bash
  cp .env.arm64.example .env
  ```


### 2. **配置环境变量**

编辑 `.env` 文件，并根据需求更新关键参数：

- `RTTYS_PASS`：Web 管理密码（必填，不填平台不启动）
- `RTTYS_TOKEN`：尚未升级固件的设备在 `RTTYS_DEVICE_PORT` 上使用的令牌（`RTTYS_LEGACY_TOKEN_ACCESS` 为 `true` 时必填）
- `TURN_USER` / `TURN_PASS`：coturn 鉴权凭据（必填，不填 coturn 不启动）

以上都没有默认值，请自行生成，例如 `openssl rand -hex 16`。如果仍在使用旧版 `.env` 示例里的值，平台启动时会给出警告。

#### **设备接入**

- `GLKVM_ACCESS_IP`：GLKVM Cloud 访问地址（留空则启动时自动检测）
- `RTTYS_DEVICE_MTLS_PORT`：新固件设备的接入端口，设备凭接入码接入（默认 `5915`）
- `DEVICE_ACCESS_HOST`：设备连接该端口使用的地址，会写入接入码（留空则使用 `GLKVM_ACCESS_IP`）。修改后（例如 IP 换成域名）已接入的设备无需改动，设备口证书会继续包含原来的地址
- `RTTYS_LEGACY_TOKEN_ACCESS`：是否继续允许旧固件设备通过 `RTTYS_DEVICE_PORT` 用 `RTTYS_TOKEN` 接入（默认 `true`）。已弃用：所有设备都改用接入码后请设为 `false`
- `SELFHOST_WEBUI_URL`：下发给接入码设备的平台地址，设备本地页面用它显示平台链接，例如 `https://www.example.com`（留空则用上面的地址加 `RTTYS_WEBUI_PORT` 拼出；平台在反向代理后面时请填写）

#### **LDAP 认证（可选）**

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

⚠️ **注意：所有配置均需在 `.env` 中完成，不需要修改 `docker-compose.yml`、模板或脚本。**

#### **OIDC 认证（可选）**

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

#### 反向代理模式（可选）

```env
REVERSE_PROXY_ENABLED=false
```

启用后（`REVERSE_PROXY_ENABLED=true`）：

- GLKVM Cloud 运行在反向代理（如 Nginx）之后
- TLS 由反向代理终止，GLKVM Cloud 内部使用 HTTP
- Web UI 与设备远程访问可共用同一个 HTTPS 端口（通常为 443）


##### 必需的反向代理请求头

反向代理必须转发以下请求头，否则可能生成包含内部端口（如 `:10443`）的访问地址：

```nginx
proxy_set_header Host                $host;
proxy_set_header X-Forwarded-Host    $host;
proxy_set_header X-Forwarded-Proto   $scheme;
proxy_set_header X-Forwarded-Port    $server_port;
proxy_set_header X-Real-IP           $remote_addr;
proxy_set_header X-Forwarded-For     $proxy_add_x_forwarded_for;
```

##### 设备远程访问域名（可选）

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
- 可与 Web UI 域名不同
- 留空时，将从 `X-Forwarded-*` 请求头自动推导

**示例：**

```text
https://www.example.com             → Web UI
https://<deviceId>.kvm.example.com  → 设备远程访问
DEVICE_ENDPOINT_HOST=kvm.example.com
```

##### 按路径访问设备（可选）

设备可以直接挂在 Web UI 的域名下访问，不再每台设备一个子域名。只需要一张证书、一条域名解析，不需要泛域名：

```text
https://www.example.com                     → Web UI
https://www.example.com/access/<deviceId>/  → 设备远程访问
```

只需配置 Nginx：使用 [`nginx-reverse-proxy-example.conf`](nginx-reverse-proxy-example.conf) 里按路径访问的 server 块。它把 `/access/<deviceId>` 转到设备访问端口，并给 Web UI 的请求加上 `X-KVM-Access-Style: path`。

- 只有用接入码接入的设备按路径访问。旧固件设备仍然使用自己的子域名 `<deviceId>.<域名>`，需要泛域名配置：有旧固件设备时请保留 `*.example.com` 的 server 块
- 同一个浏览器可以同时打开多台设备
- 必须在前面配 Nginx，通过 IP 访问平台时不可用


⚠️ **注意：所有配置均需在 `.env` 中完成，不需要修改 `docker-compose.yml`、模板或脚本。**

### 3. **启动服务**

```bash
docker-compose up -d
```

如果你修改了 `.env` 或模板文件，请重新加载服务：

```bash
docker-compose down && docker-compose up -d
```

### 4. **访问平台**

安装完成后，通过以下地址访问平台：

```bash
https://<你的服务器公网 IP>
```

