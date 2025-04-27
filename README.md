<!-- TOC -->

- [1. 流程](#1-流程)
- [2. diyNAS](#2-diynas)
    - [2.1. Linux](#21-linux)
    - [2.2. Samba](#22-samba)
    - [2.3. Syching](#23-syching)
    - [2.4. Nginx](#24-nginx)
    - [2.5. Gitea](#25-gitea)
    - [2.6. Frp](#26-frp)
- [3. docs](#3-docs)

<!-- /TOC -->
# 1. 流程
```mermaid
flowchart TD
    subgraph 公网用户
        A["用户设备"] -->|1- Git SSH 加密连接| B["公网服务器:22"]
        A -->|2- Syncthing P2P/中继| C["公网服务器:22000"]
    end

    subgraph 公网服务器
        B -->|SSH 隧道解密| D["反向代理 Nginx"]
        D -->|3- 转发到本地端口| E["Localhost:3000 [SSH 隧道入口]"]
        C -->|中继加密流量| F["Syncthing 中继节点"]
    end

    subgraph 局域网NAS
        E -->|SSH 加密通道| G["Gitea:3000"]
        F -->|P2P 直连或中继| H["Syncthing:22000"]
        G -->|数据同步| I["/var/lib/gitea"]
        H -->|文件同步| J["/srv/nas/share"]
        J -->|SMB/NFS| K["内网设备"]
    end

    style A stroke:#333,stroke-width:2px
    style B stroke:#d33,stroke-width:2px
    style C stroke:#393,stroke-width:2px
    style G stroke:#06c,stroke-width:2px
    style H stroke:#690,stroke-width:2px
```
# 2. diyNAS
DIY NAS with Linux Samba Syncthing Nginx Gitea Frp
## 2.1. Linux

## 2.2. Samba

## 2.3. Syching

## 2.4. Nginx

## 2.5. Gitea 

## 2.6. Frp

# 3. docs