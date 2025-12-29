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

- `RTTYS_TOKEN`：设备连接令牌（留空则使用默认值）
- `RTTYS_PASS`：Web 管理密码（留空则使用默认值 **StrongP@ssw0rd**）
- `TURN_USER` / `TURN_PASS`：coturn 鉴权凭据（留空则使用默认值）
- `GLKVM_ACCESS_IP`：GLKVM Cloud 访问地址（留空则启动时自动检测）

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

#### **反向代理模式（可选）**

```env
# 启用反向代理模式（例如在 GLKVM Cloud 前使用 Nginx）
# 启用后，TLS 由反向代理终止，GLKVM Cloud 内部使用明文 HTTP
REVERSE_PROXY_ENABLED=false
```

当 `REVERSE_PROXY_ENABLED` 设置为 `true` 时，GLKVM Cloud 将运行在 **反向代理（如 Nginx）之后**：

- HTTPS 证书由反向代理管理（而不是由 GLKVM Cloud 本身管理）
- GLKVM Cloud 内部以明文 HTTP 方式监听
- 同一个 HTTPS 端口可同时用于：
  - 访问 GLKVM Cloud Web 管理界面
  - 访问远程 KVM 设备

例如，在正确配置 Nginx 的情况下：

```nginx
# 转发原始的主机名、协议、端口以及客户端 IP
# 在反向代理模式下，这些 Header 是必须的
proxy_set_header Host                $host;
proxy_set_header X-Forwarded-Host    $host;
proxy_set_header X-Forwarded-Proto   $scheme;
proxy_set_header X-Forwarded-Port    $server_port;
proxy_set_header X-Real-IP           $remote_addr;
proxy_set_header X-Forwarded-For     $proxy_add_x_forwarded_for;
```

你可以通过以下地址访问：

```text
https://www.example.com            → GLKVM Cloud 管理界面
https://<device_id>.example.com    → 远程设备访问
```

这两个地址可以共用 **同一个 HTTPS 端口（443）**，由反向代理根据访问的域名进行路由区分。

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

