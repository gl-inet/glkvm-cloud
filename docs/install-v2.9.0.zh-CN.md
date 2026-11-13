# 安装 GLKVM Cloud 2.9.0

[English](install-v2.9.0.md)

适用于使用旧固件、通过复制脚本接入自部署云的 GL KVM 设备

## 1. 安装平台

准备一台未安装 GLKVM Cloud 的 Debian / Ubuntu 或 Rocky Linux / AlmaLinux 服务器，支持 x86_64 和 aarch64

在防火墙和云安全组放行 **443/TCP、10443/TCP、5912/TCP、3478/TCP 和 UDP**，服务器需能访问下载站和系统软件源

通过 SSH 登录**服务器**，执行：

```sh
curl -fSL https://kvm-cloud.gl-inet.com/selfhost/v2.9.0/install.sh -o install-v2.9.0.sh
sudo bash install-v2.9.0.sh
```

使用 root 登录时可省略 `sudo`。脚本会自动安装所需组件，固定部署 2.9.0，镜像从下载站获取，无需访问 Docker Hub

公网 IP 会自动检测。内网部署或需要指定地址时，将第二条命令替换为下面这条，把示例 IP 换成设备能访问的服务器地址：

```sh
sudo bash install-v2.9.0.sh --ip 192.168.1.100
```

## 2. 登录平台

安装完成后，终端会显示平台地址和管理员密码。浏览器打开该地址，使用 `admin` 和输出的密码登录

默认使用自签证书，浏览器会提示证书不受信任，确认是自己的服务器地址后继续访问

## 3. 添加设备

1. 在平台点“添加设备”，选择 KVM，复制生成的脚本
2. 通过本地网页终端或 SSH 登录 **KVM 设备**，粘贴并执行脚本
3. 返回平台，设备在线后即可打开远程控制
