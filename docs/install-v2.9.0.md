# Install GLKVM Cloud 2.9.0

[中文](install-v2.9.0.zh-CN.md)

Use this version for GL KVM devices running older firmware that connect to a self-hosted platform using a copied script.

## 1. Install the platform

Prepare a Debian / Ubuntu or Rocky Linux / AlmaLinux server without an existing GLKVM Cloud installation. Both x86_64 and aarch64 are supported.

Allow **443/TCP, 10443/TCP, 5912/TCP, and 3478/TCP and UDP** in the firewall and cloud security group. The server must be able to reach the download site and Docker Hub.

Connect to the **server** over SSH and run:

```sh
curl -fSL https://kvm-cloud.gl-inet.com/selfhost/v2.9.0/install.sh -o install-v2.9.0.sh
sudo bash install-v2.9.0.sh
```

You can omit `sudo` when logged in as root. The script installs the required components and deploys version 2.9.0.

The public IP is detected automatically. For a LAN deployment or to specify an address, replace the second command with the following. Replace the example IP with the server address your devices can reach:

```sh
sudo bash install-v2.9.0.sh --ip 192.168.1.100
```

## 2. Sign in

When installation finishes, the terminal displays the platform URL and administrator password. Open the URL in your browser and sign in with `admin` and the displayed password.

The platform uses a self-signed certificate by default. If your browser displays a certificate warning, confirm that the address belongs to your server before continuing.

## 3. Add a device

1. Click **Add Device** on the platform, select KVM, and copy the generated script
2. Open the **KVM device's** local web terminal or connect to it over SSH, then paste and run the script
3. Return to the platform. Once the device is online, open its remote control page
