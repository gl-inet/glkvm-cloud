# Deploy with Docker Compose

This guide deploys GLKVM Cloud by hand with the Docker Compose files in this directory. For the one-line installer, the ports to open and upgrades, see the [main README](../README.md).

You need Docker and Docker Compose 1.27 or later, or the `docker compose` plugin, on the server. On Ubuntu 22.04 or later and Debian 12, `apt install -y docker.io docker-compose` installs both; on other systems, follow [Docker's installation guide](https://docs.docker.com/engine/install/).

1. **Clone the repository and prepare the environment template**

   ```bash
   git clone https://github.com/gl-inet/glkvm-cloud.git
   cd glkvm-cloud/docker-compose/
   ```

   - For x86_64 (amd64):

     ```bash
     cp .env.example .env
     ```
   - For arm64 (AArch64):

     ```bash
     cp .env.arm64.example .env
     ```

   The two templates differ only in the images (`GLKVM_IMAGE`, `COTURN_IMAGE`).

2. **Configure environment variables**

   Edit `.env` and set the required parameters:

   - `RTTYS_PASS`: admin password (required: the platform does not start without it)
   - `RTTYS_TOKEN`: token of devices added with the old script, on `RTTYS_DEVICE_PORT` (required while `RTTYS_LEGACY_TOKEN_ACCESS` is `true`; a new installation without such devices can turn that off instead)
   - `TURN_USER` / `TURN_PASS`: coturn authentication credentials (required: coturn does not start without them)

   None of them has a default. Generate your own, e.g. with `openssl rand -hex 16`. The platform warns at startup while one of them is still an example value from older `.env` examples.

   **Admin Account:**

   - `RTTYS_ADMIN_NAME`: admin username, letters and digits only (default `admin`). Its password is `RTTYS_PASS`

   **Device Access:**

   - `GLKVM_ACCESS_IP`: the server's public address. TURN uses it, and so do access codes unless `DEVICE_ACCESS_HOST` is set. Leave empty to detect the public IP at startup; on an internal network, set the server's internal IP
   - `RTTYS_DEVICE_MTLS_PORT`: port for devices added with an access code (default `5915`)
   - `DEVICE_ACCESS_HOST`: address in access codes that devices connect to on that port, an IP address or a domain name (leave empty to use `GLKVM_ACCESS_IP`). Devices that already joined keep working after it changes
   - `RTTYS_LEGACY_TOKEN_ACCESS`: keep accepting devices added with the old script, which use `RTTYS_TOKEN` on `RTTYS_DEVICE_PORT` (default `true`). Deprecated: set it to `false` once all devices use access codes, or from the start on a new installation; then `RTTYS_TOKEN` and `RTTYS_DEVICE_PORT` are not needed
   - `SELFHOST_WEBUI_URL`: web UI address sent to devices that joined with an access code, for their local web page to link to, e.g. `https://www.example.com`. Set it behind a reverse proxy. If empty, it is built from `WEB_UI_HOST`, `DEVICE_ACCESS_HOST` or `GLKVM_ACCESS_IP`, the first one set, with `RTTYS_WEBUI_PORT`

   **Ports:**

   - `RTTYS_WEBUI_PORT`: web UI (default `443`)
   - `RTTYS_HTTP_PROXY_PORT`: remote access to devices (default `10443`)
   - `RTTYS_DEVICE_PORT`: devices added with the old script (default `5912`)
   - `TURN_PORT`: TURN server, TCP and UDP (default `3478`)

   Allow these ports and `RTTYS_DEVICE_MTLS_PORT` in the cloud security group and firewall. In [reverse proxy mode](#reverse-proxy-mode-optional), keep `RTTYS_WEBUI_PORT` and `RTTYS_HTTP_PROXY_PORT` closed instead.

   **Web UI Domain (Optional):**

   - `WEB_UI_HOST`: only accept the web UI and API on this domain, e.g. `www.example.com` (leave empty to accept any)

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
   - `LDAP_ADMIN_GROUP` / `LDAP_ADMIN_USERS`: comma-separated groups and usernames that get the admin role (optional). Other LDAP users get the Operator role

   **OIDC Authentication (Optional):**

   - `OIDC_ENABLED`: set to `true` to enable OIDC authentication (default: `false`)
   - `OIDC_PRIMARY_LOGIN`: set to `true` to offer OIDC first on the login page; the username and password form moves behind a link (default: `false`)
   - `OIDC_ISSUER`: OIDC issuer URL provided by your identity provider
      (e.g. `https://accounts.google.com`, `https://your-tenant.auth0.com/`)
   - `OIDC_CLIENT_ID`: client ID issued by your OIDC provider
   - `OIDC_CLIENT_SECRET`: client secret issued by your OIDC provider
   - `OIDC_AUTH_URL`: authorization endpoint URL
   - `OIDC_TOKEN_URL`: token endpoint URL
   - `OIDC_REDIRECT_URL`: redirect (callback) URL registered in your OIDC provider
      Any domain, but the path must be `/auth/oidc/callback`
      Example: `https://your-domain.example.com/auth/oidc/callback`
   - `OIDC_SCOPES`: space-separated list of requested scopes (default: `"openid profile email"`)
   - `OIDC_ALLOWED_USERS`: comma-separated list of allowed emails or domains (optional)
      Example: `user@example.com,@example.com`
   - `OIDC_ALLOWED_SUBS`: comma-separated list of allowed OIDC subject (`sub`) IDs (optional)
   - `OIDC_ALLOWED_USERNAMES`: comma-separated list of allowed usernames (`preferred_username` or `name`) (optional)
   - `OIDC_ALLOWED_GROUPS`: comma-separated list of allowed OIDC groups (optional)
   - `OIDC_ADMIN_GROUP` / `OIDC_ADMIN_USERS`: comma-separated groups, and users matched by `preferred_username` or email, that get the admin role (optional). Other OIDC users get the Operator role

   All configuration is done in `.env`: you don't need to modify `docker-compose.yml`, templates or scripts.

3. **Add your own certificate (optional)**

   Put the certificate (full chain) and its private key in `certificate/` as `glkvm.cer` and `glkvm.key`, in PEM format (for Let's Encrypt, `fullchain.pem` and `privkey.pem`). Without them, the platform generates one on first start that browsers do not trust (see [`certificate/README.md`](certificate/README.md)). For the domain and DNS setup, see [Use Your Own Domain and SSL Certificate](../README.md#use-your-own-domain-and-ssl-certificate-optional). In reverse proxy mode, the certificate goes to Nginx instead.

4. **Start the services**

   ```bash
   docker-compose up -d
   ```

   After changing `.env`, templates or the certificate, apply the changes:

   ```bash
   docker-compose down && docker-compose up -d
   ```

   With the Docker CLI plugin, use `docker compose` instead of `docker-compose`.

   Check that both services are up with `docker-compose ps`. If one keeps restarting, `docker-compose logs rttys coturn` shows why, usually a required setting left empty.

5. **Sign in**

   Open the platform and sign in as `admin` (or `RTTYS_ADMIN_NAME`) with `RTTYS_PASS`:

   ```text
   https://<your_server_public_ip>
   ```

   Opening it by IP address triggers a browser certificate warning.

   Back up `database/`, for example with `cp -a database ../database-backup`. It holds the database and `pki`, the CA trusted by devices that joined with an access code. To upgrade, see [Upgrade](../README.md#upgrade).

## Reverse Proxy Mode (Optional)

GLKVM Cloud can run behind a reverse proxy such as Nginx, which terminates TLS. The web UI and remote access to devices then share one HTTPS port (usually 443), and GLKVM Cloud serves plain HTTP behind it.

1. **Set in `.env`:**

   ```env
   REVERSE_PROXY_ENABLED=true
   RTTYS_WEBUI_PORT=1443
   SELFHOST_WEBUI_URL=https://www.example.com
   ```

   - `RTTYS_WEBUI_PORT` moves the web UI off port 443, which Nginx takes. `1443` matches the Nginx example
   - `SELFHOST_WEBUI_URL` is the public address of the web UI. Without it, devices are given the internal port
   - Devices still connect to `RTTYS_DEVICE_MTLS_PORT` and `TURN_PORT` directly. Don't put them behind Nginx: the device port checks each device's factory certificate itself

2. **Keep the plain HTTP ports closed.** `RTTYS_WEBUI_PORT` and `RTTYS_HTTP_PROXY_PORT` serve plain HTTP, and Docker publishes them on all interfaces, past host firewalls such as ufw. Don't allow them in the cloud security group, so they are only reached through Nginx.

3. **Configure Nginx** from [`nginx-reverse-proxy-example.conf`](nginx-reverse-proxy-example.conf): replace `example.com` with your domain and `/path/to/` with your certificate files. As it is, the example opens devices on a subdomain of their own, `https://<deviceId>.example.com`, which needs a wildcard DNS record and certificate. To open them by path under the web UI's domain instead, see [Path-Based Device Access](#path-based-device-access-optional).

### Required Reverse Proxy Headers

The reverse proxy must forward the following headers, or GLKVM Cloud may generate URLs with internal ports (e.g. `:10443`), and pass WebSocket upgrades on (`$connection_upgrade` comes from the `map` at the top of the Nginx example):

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

### Device Remote Access Domain (Optional)

```env
DEVICE_ENDPOINT_HOST=
```

- Only used when `REVERSE_PROXY_ENABLED=true`
- The domain for remote access to devices, whose addresses then look like:

```text
https://<deviceId>.<DEVICE_ENDPOINT_HOST>/
```

**Notes:**

- Do not include the scheme (`http://` or `https://`)
- Do not include any path
- The domain may differ from the Web UI domain. Devices are opened on its subdomains, so it needs its own wildcard DNS record and certificate, e.g. `*.kvm.example.com`
- If empty, the host and port come from the `X-Forwarded-*` headers

**Example:**

```text
https://www.example.com             → Web UI
https://<deviceId>.kvm.example.com  → Device remote access
DEVICE_ENDPOINT_HOST=kvm.example.com
```

### Path-Based Device Access (Optional)

Devices can be opened under the Web UI's own domain instead of a host of their own, so one certificate and one DNS record are enough, no wildcard:

```text
https://www.example.com                     → Web UI
https://www.example.com/access/<deviceId>/  → Device remote access
```

Besides reverse proxy mode above, it takes one line in Nginx: in the web UI block of [`nginx-reverse-proxy-example.conf`](nginx-reverse-proxy-example.conf), change the value of `X-KVM-Access-Style` from `""` to `path` and reload Nginx. The block already serves `/access/<deviceId>` from the device access port; the header tells the platform to open devices there.

- Only devices added with an access code are opened by path. Devices added with the old script keep their own host `<deviceId>.<domain>`, which still needs the wildcard setup: keep the `*.example.com` server block while you have them
- Several devices can be open in one browser at the same time
- It needs Nginx in front, so it is not available when the platform is accessed by IP
