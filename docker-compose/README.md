# Quick Start

 This guide shows how to deploy **glkvm-cloud** using the provided Docker Compose environment template. 

1. **Clone the repository and prepare the environment template**

   ```bash
   git clone https://github.com/gl-inet/glkvm-cloud.git
   cd glkvm-cloud/docker-compose/
   ```
* For **x86_64 (amd64)**:

  ```bash
  cp .env.example .env
  ```
* For **arm64 (AArch64)**:
  ```bash
  cp .env.arm64.example .env
  ```

2. **Configure environment variables**

   Edit `.env` and update the required parameters: 

   - `RTTYS_TOKEN`: device connection token (leave empty to use the default)
   - `RTTYS_PASS`: web management password (leave empty to use the default **StrongP@ssw0rd**)
   - `TURN_USER` / `TURN_PASS`: coturn authentication credentials (leave empty to use the default)
   - `GLKVM_ACCESS_IP`: glkvm cloud access address (leave empty to auto-detect at startup)
   - `RTTYS_DEVICE_MTLS_PORT`: device port for new firmware, which joins with an access code (default `5915`)
   - `DEVICE_ACCESS_HOST`: address devices dial for that port, put into access codes (leave empty to use `GLKVM_ACCESS_IP`)
   - `RTTYS_LEGACY_TOKEN_ACCESS`: keep accepting old-firmware devices with `RTTYS_TOKEN` on `RTTYS_DEVICE_PORT` (default `true`)

   **LDAP Authentication (Optional):**

   - `LDAP_ENABLED`: set to `true` to enable LDAP authentication (default: `false`)
   - `LDAP_SERVER`: LDAP server hostname or IP address
   - `LDAP_PORT`: LDAP server port (default: `389`, for TLS use `636`)
   - `LDAP_USE_TLS`: set to `true` to enable TLS encryption (default: `false`)
   - `LDAP_BIND_DN`: service account distinguished name
   - `LDAP_BIND_PASSWORD`: service account password
   - `LDAP_BASE_DN`: search base for user queries
   - `LDAP_USER_FILTER`: LDAP query filter (default: `(uid=%s)`)
   - `LDAP_ALLOWED_GROUPS`: comma-separated list of authorized groups (optional)
   - `LDAP_ALLOWED_USERS`: comma-separated list of authorized users (optional)

   ⚠️ **Note:** All configuration should be done in the `.env` file.
    You don’t need to modify `docker-compose.yml`, templates, or scripts directly.

   **OIDC Authentication (Optional):**

   - `OIDC_ENABLED`: set to `true` to enable OIDC authentication (default: `false`)
   - `OIDC_ISSUER`: OIDC issuer URL provided by your identity provider
      (e.g. `https://accounts.google.com`, `https://your-tenant.auth0.com/`)
   - `OIDC_CLIENT_ID`: client ID issued by your OIDC provider
   - `OIDC_CLIENT_SECRET`: client secret issued by your OIDC provider
   - `OIDC_AUTH_URL`: authorization endpoint URL
   - `OIDC_TOKEN_URL`: token endpoint URL
   - `OIDC_REDIRECT_URL`: redirect (callback) URL registered in your OIDC provider
      Domain is user-defined, but the path must be fixed: `/auth/oidc/callback`
      Example: `https://your-domain.example.com/auth/oidc/callback`
   - `OIDC_SCOPES`: space-separated list of requested scopes (default: `"openid profile email"`)
   - `OIDC_ALLOWED_USERS`: comma-separated list of allowed emails or domains (optional)
      Example: `user@example.com,@example.com`
   - `OIDC_ALLOWED_SUBS`: comma-separated list of allowed OIDC subject (`sub`) IDs (optional)
   - `OIDC_ALLOWED_USERNAMES`: comma-separated list of allowed usernames (`preferred_username` or `name`) (optional)
   - `OIDC_ALLOWED_GROUPS`: comma-separated list of allowed OIDC groups (optional)
   

#### Reverse Proxy Mode (Optional)

```env
REVERSE_PROXY_ENABLED=false
```

When enabled (`REVERSE_PROXY_ENABLED=true`):

- GLKVM Cloud runs behind a reverse proxy (e.g. Nginx)
- TLS is terminated at the reverse proxy; GLKVM Cloud uses plain HTTP internally
- The Web UI and remote device access can share the same HTTPS port (usually 443)


##### Required Reverse Proxy Headers

The reverse proxy **must** forward the following headers; otherwise, GLKVM Cloud may generate URLs containing internal ports (e.g. `:10443`):

```nginx
proxy_set_header Host                $host;
proxy_set_header X-Forwarded-Host    $host;
proxy_set_header X-Forwarded-Proto   $scheme;
proxy_set_header X-Forwarded-Port    $server_port;
proxy_set_header X-Real-IP           $remote_addr;
proxy_set_header X-Forwarded-For     $proxy_add_x_forwarded_for;
```


##### Device Remote Access Domain (Optional)

```env
DEVICE_ENDPOINT_HOST=
```

- **Effective only when** `REVERSE_PROXY_ENABLED=true`
- Used to specify the domain for device remote access
- Device access URLs are generated as:

```text
https://<deviceId>.<DEVICE_ENDPOINT_HOST>/
```

**Notes:**

- Do not include the scheme (`http://` or `https://`)
- Do not include any path
- The domain may differ from the Web UI domain
- If left empty, the host/port will be derived from `X-Forwarded-*` headers

**Example:**

```text
https://www.example.com             → Web UI
https://<deviceId>.kvm.example.com  → Device remote access
DEVICE_ENDPOINT_HOST=kvm.example.com
```





⚠️ **Note:** All configuration should be done in the `.env` file.
    You don’t need to modify `docker-compose.yml`, templates, or scripts directly.

3. **Start the services**

   ```bash
   docker-compose up -d
   ```

   If you modify `.env` or template files, make sure to apply the updates:

   ```bash
   docker-compose down && docker-compose up -d
   ```

4. **Platform Access**

   Once the installation is complete, access the platform via: 

    ```bash
    https://<your_server_public_ip>
    ```
