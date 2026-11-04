# 自部署轻量云 KVM 远程管理平台

[English Documentation](./README.md) | 中文

GLKVM 轻量云（GLKVM Cloud）是一个部署在自己服务器上的 GL.iNet KVM 远程管理平台，面向个人和小微企业，基于 [rttys](https://github.com/zhaojh329/rttys) 开发。数据和带宽都在你自己的服务器上。

## 主要功能

* **用户组和设备组** - 为用户组分配设备组，让不同用户管理不同设备
* **接入码接入** - 在设备本地网页粘贴接入码即可添加设备，设备凭出厂证书证明身份
* **KVM 终端** - 在网页上登录 KVM 设备本身的终端
* **远程控制** - 在浏览器里远程控制 KVM 的桌面
* **批量执行命令** - 一次对多台设备执行命令
* **LDAP 和 OIDC 登录** - 用公司的目录服务或身份提供商登录
* **部署灵活** - 内网或公网都能部署，支持 x86_64 和 arm64

## 自部署指南

以下操作系统已通过测试：

### Debian 系列

* Ubuntu 22.04 / 24.04
* Debian 12

### Red Hat 系列

* AlmaLinux 8 / 9
* Rocky Linux 8 / 9
* CentOS Stream 9

Ubuntu 20.04、Debian 11 等较老的系统自带的 Docker Compose 读不了部署文件。在这些系统上请用 Docker 手动安装，并使用 Docker Compose 1.27 或以上版本，或 `docker compose` 插件。

### 系统要求

|     组件     | 最低配置要求 |
| :----------: | :----------: |
|     CPU      |  1 核及以上  |
|     内存     |    ≥ 1 GB    |
|     存储     |   ≥ 40 GB    |
|   网络带宽   |   ≥ 3 Mbps   |
| KVM 固件版本 |  ≥ v1.11.0   |

### 云安全组端口要求

如果你的服务器提供商（如 AWS、阿里云等）启用了云安全组，请确保以下端口已开放：

| 端口  | 协议    | 用途                                   |
| ----- | ------- | -------------------------------------- |
| 443   | TCP     | 平台 Web 界面                          |
| 10443 | TCP     | 设备远程访问（远程控制）               |
| 5912  | TCP     | 用旧脚本接入的设备（已弃用，见[旧固件设备](#旧固件设备)）；新安装不需要 |
| 5915  | TCP     | 用接入码接入的设备                     |
| 3478  | TCP/UDP | WebRTC TURN 中继                       |

这些端口不能被服务器上的其他程序占用，否则平台可能无法启动。

### 安装

有两种安装方式。

#### A) 一键安装脚本（推荐，仅支持 x86_64 / amd64）

安装脚本会自动安装 Docker 和 Docker Compose、下载镜像并启动服务。请用 root 运行：

```bash
( command -v curl >/dev/null 2>&1 && curl -fsSL https://kvm-cloud.gl-inet.com/selfhost/install.sh || wget -qO- https://kvm-cloud.gl-inet.com/selfhost/install.sh ) | sudo bash
```

如果系统里没有 `sudo`，把命令末尾的 `sudo` 去掉即可。

安装脚本会把所有文件放在当前目录下的 `glkvm_cloud` 里，例如 `/root/glkvm_cloud`。下文用 `~/glkvm_cloud` 指代这个目录。

#### B) 使用 Docker 手动安装

支持 x86_64（amd64）和 arm64（AArch64），见 [`docker-compose/README-CN.md`](docker-compose/README-CN.md)。

### 平台访问

安装完成后，安装脚本会在控制台输出平台地址和管理员的用户名、密码，例如：

```text
   🌐 You can now access the GLKVM Cloud platform at:
       https://<你的服务器公网IP>

   👤 Admin username:
       admin

   🔑 Admin password:
       <自动生成的密码>
```

通过 IP 访问时，浏览器会提示证书不受信任。如需消除该提示，请使用自己的域名和证书，见[使用自有域名和 SSL 证书](#使用自有域名和-ssl-证书可选)。

如果页面打不开，用 `cd ~/glkvm_cloud && docker-compose ps` 检查两个服务是否都在运行。如果某个服务一直在重启，用 `docker-compose logs rttys coturn` 查看原因。

在内网部署时，请把 `~/glkvm_cloud/.env` 里的 `GLKVM_ACCESS_IP` 改成服务器的内网 IP，然后[重启服务](#重启服务)。否则平台会使用自己的公网 IP，内网里的设备可能连不上。

### 升级

升级时部署文件和镜像要一起更新。只改镜像版本号不够：v3.0 之前的 `docker-compose.yml` 没有对外开放 5915 端口，设备无法用接入码接入（添加设备弹窗会给出提示）。

**用一键安装脚本安装的：**

1. 备份数据库：

   ```bash
   cp -a ~/glkvm_cloud/database ~/glkvm_cloud-database-backup
   ```

2. 用 root 在 `glkvm_cloud` 所在的目录（不是进到 `glkvm_cloud` 里面）重新运行安装脚本。脚本会下载新镜像和部署文件，保留 `.env` 和数据库，并重启服务。
3. 在云安全组放行 5915/TCP。

**手动安装的：** 以下操作都在 `docker-compose.yml` 所在的目录里进行。

1. 按上面的方法备份 `database`。
2. 更新部署文件：用 git clone 安装的执行 `git pull`，否则下载并解压部署包。两种方式都不会改动 `.env` 和数据库：

   ```bash
   curl -fsSLO https://kvm-cloud.gl-inet.com/selfhost/docker-compose.tar.gz
   tar -xzf docker-compose.tar.gz
   ```

3. 检查 `.env`。从 v3.0 起，`RTTYS_PASS`、`TURN_USER`、`TURN_PASS` 没有默认值，开启旧 token 接入（`RTTYS_LEGACY_TOKEN_ACCESS`）时 `RTTYS_TOKEN` 也没有默认值，任一项为空时平台或 coturn 不会启动。如果之前这几项留空，旧固件设备用的是原来的默认值。要让这些设备在迁移到接入码之前继续可用，请填上：

   ```env
   RTTYS_TOKEN=DeviceTokenYouCanChangeMe
   TURN_USER=glkvmcloudwebrtcuser
   TURN_PASS=AnotherS3cret
   ```

4. 在云安全组和防火墙放行 5915/TCP。
5. 获取新镜像并重启：

   ```bash
   docker-compose pull    # 或：curl -fsSLO https://kvm-cloud.gl-inet.com/selfhost/glkvmcloud.tar && docker load -i glkvmcloud.tar
   docker-compose down && docker-compose up -d
   ```

   使用 Docker CLI 插件时，把 `docker-compose` 换成 `docker compose`。

## 使用

### 添加 KVM 设备

设备固件需要 v1.11.0 或更高版本。如果要用自己的域名，请先完成[域名配置](#使用自有域名和-ssl-证书可选)，这样接入码里写的就是域名。

1. 在平台的设备页面打开「添加设备」，生成接入码并复制。每个接入码 24 小时内有效，只能用一次，哪台设备都能用。

   ![添加设备](img/adddevice.png)

2. 在浏览器打开设备的本地网页（`https://<设备IP>`），进入「自部署云」页面，粘贴接入码并保存。如果设备已接入其他自部署平台，要先在这个页面断开。
3. 设备接入成功后，会出现在设备列表里：

   ![设备列表](img/devicelist.png)

设备用接入码里的 CA 校验平台，平台用设备的出厂证书校验设备。

### 旧固件设备

用旧脚本接入的设备，在平台升级后继续通过 5912 端口连接。低于 v1.11.0 的固件已不能再新接入，在设备终端执行脚本的接入方式也不再支持。迁移到接入码的步骤：

1. 先升级平台，再把设备固件升级到 v1.11.0 或更高版本。
2. 在设备的「自部署云」页面粘贴接入码。设备在平台上的记录、ID 和分组都会保留；如果设备仍以旧方式在线，切换过程中不会掉线。

5912 端口已弃用，后续版本会移除。没有设备再使用后，可设置 `RTTYS_LEGACY_TOKEN_ACCESS=false` 关闭。没有这类设备的新安装，一开始就可以关闭。

### KVM 终端

![KVM 终端截图](img/ssh.png)

### 远程控制

![远程控制截图](img/web.png)

## 使用自有域名和 SSL 证书（可选）

如果只是想体验 GLKVM 轻量云，并且不介意浏览器的证书警告，可以跳过本节，继续用服务器公网 IP 访问平台。

生产环境建议用自己的域名和有效证书访问平台。设备会通过子域名打开，例如 `https://<设备ID>.your-domain.com:10443`，因此需要泛域名解析和泛域名证书，配置方法见下文。如果平台前面有 Nginx，也可以改为按路径在平台域名下打开设备，只需要一张单域名证书，见[按路径访问设备](docker-compose/README-CN.md#按路径访问设备可选)。

### 添加 DNS 记录

| 主机记录 | 类型 | 值                | 用途          |
| -------- | ---- | ----------------- | ------------- |
| `www`    | A    | 你的服务器公网 IP | 平台 Web 界面 |
| `*`      | A    | 你的服务器公网 IP | 设备远程访问  |

平台要用 `www` 这样的子域名访问，不能用根域名：设备地址是把平台域名的第一段换成设备 ID，例如 `www.your-domain.com` 对应 `<设备ID>.your-domain.com`。

### 使用自定义 SSL 证书

申请一张 `*.your-domain.com` 的泛域名证书，它同时覆盖 `www.your-domain.com` 和各设备的子域名。Let's Encrypt 的泛域名证书只能用 DNS 验证申请。

把证书（完整证书链）和私钥放到 `~/glkvm_cloud/certificate`，都用 PEM 格式，文件名必须是：

* `glkvm.cer`（Let's Encrypt 的 `fullchain.pem`）
* `glkvm.key`（Let's Encrypt 的 `privkey.pem`）

没有放证书时，GLKVM 轻量云会在首次启动时自动生成一张证书，浏览器会提示不受信任。证书续期后，把新文件放进去并重启服务。

设备连接的端口（5912 和 5915）不使用这张证书，而是使用平台自有 CA 签发的证书，CA 在 `~/glkvm_cloud/database/pki`。请保留该目录并纳入备份：用接入码接入的设备信任这个 CA，CA 被替换后设备将无法接入。

### 接入码使用域名（可选）

接入码默认让设备连接 `GLKVM_ACCESS_IP`。如果希望设备通过域名连接，服务器 IP 变了也不受影响，在 `~/glkvm_cloud/.env` 中设置：

```env
DEVICE_ACCESS_HOST=www.your-domain.com
```

重启后新生成的接入码才会使用域名，已经接入的设备不受影响。

### 重启服务

修改证书或 `.env` 后，需要重启 GLKVM 轻量云：

```bash
cd ~/glkvm_cloud
docker-compose down && docker-compose up -d
```

使用 Docker CLI 插件时，把 `docker-compose` 换成 `docker compose`。之后即可通过 `https://www.your-domain.com` 访问平台。
