package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// 颜色定义
const (
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Reset  = "\033[0m"
)

// 检查 root 权限
func checkRoot() {
	if os.Getuid() != 0 {
		log.Fatal(Red + "错误：请使用 root 用户运行此程序！" + Reset)
	}
}

// 执行 Shell 命令
func runCmd(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// 安装基础依赖
func installDeps() {
	fmt.Println(Yellow + "[1/6] 安装系统依赖..." + Reset)
	runCmd("apt", "update")
	runCmd("apt", "install", "-y", "curl", "wget", "git", "ufw", "nginx", "python3-certbot-nginx", "samba")
}

// 配置 Samba
func setupSamba() {
	fmt.Println(Yellow + "[2/6] 配置 Samba (内网共享)..." + Reset)
	os.MkdirAll("/srv/nas/share", 0777)

	// 写入 smb.conf
	smbConf := `[global]
   workgroup = WORKGROUP
   security = user

[share]
   path = /srv/nas/share
   browsable = yes
   writable = yes
   guest ok = no
   valid users = ` + os.Getenv("SUDO_USER") + "\n"

	os.WriteFile("/etc/samba/smb.conf", []byte(smbConf), 0644)

	// 设置 Samba 密码
	fmt.Print(Green + "设置 Samba 密码（用于访问共享）：" + Reset)
	runCmd("smbpasswd", "-a", os.Getenv("SUDO_USER"))
	runCmd("systemctl", "restart", "smbd", "nmbd")
	runCmd("ufw", "allow", "445/tcp")
}

// 安装 Gitea
func setupGitea() {
	fmt.Println(Yellow + "[3/6] 安装 Gitea (代码托管)..." + Reset)
	runCmd("wget", "-O", "/usr/local/bin/gitea", "https://dl.gitea.io/gitea/latest/gitea-linux-amd64")
	runCmd("chmod", "+x", "/usr/local/bin/gitea")
	runCmd("useradd", "-r", "-d", "/var/lib/gitea", "-m", "-s", "/bin/bash", "gitea")

	// 配置 systemd 服务
	giteaService := `[Unit]
Description=Gitea
After=network.target

[Service]
User=gitea
WorkingDirectory=/var/lib/gitea
ExecStart=/usr/local/bin/gitea web -c /etc/gitea/app.ini
Restart=always

[Install]
WantedBy=multi-user.target`

	os.WriteFile("/etc/systemd/system/gitea.service", []byte(giteaService), 0644)
	runCmd("systemctl", "enable", "--now", "gitea")
	runCmd("ufw", "allow", "3000/tcp")
}

// 安装 Syncthing
func setupSyncthing() {
	fmt.Println(Yellow + "[4/6] 安装 Syncthing (文件同步)..." + Reset)
	runCmd("curl", "-s", "https://syncthing.net/release-key.txt", "|", "apt-key", "add", "-")
	os.WriteFile("/etc/apt/sources.list.d/syncthing.list", []byte("deb https://apt.syncthing.net/ syncthing stable"), 0644)
	runCmd("apt", "update")
	runCmd("apt", "install", "-y", "syncthing")

	// 配置 systemd 服务
	syncthingService := `[Unit]
Description=Syncthing for %i
After=network.target

[Service]
User=%i
ExecStart=/usr/bin/syncthing serve --no-browser --no-restart --logfile=default
Restart=on-failure

[Install]
WantedBy=multi-user.target`

	os.WriteFile("/etc/systemd/system/syncthing@.service", []byte(syncthingService), 0644)
	runCmd("systemctl", "enable", "--now", "syncthing@"+os.Getenv("SUDO_USER"))
	runCmd("ufw", "allow", "8384/tcp")
	runCmd("ufw", "allow", "22000/tcp")
}

// 配置 Nginx 和 Certbot
func setupNginx() {
	fmt.Println(Yellow + "[5/6] 配置 Nginx (HTTPS 代理)..." + Reset)
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入 Gitea 域名（如 git.example.com）：")
	giteaDomain, _ := reader.ReadString('\n')
	giteaDomain = strings.TrimSpace(giteaDomain)

	fmt.Print("请输入 Syncthing 域名（如 sync.example.com）：")
	syncthingDomain, _ := reader.ReadString('\n')
	syncthingDomain = strings.TrimSpace(syncthingDomain)

	// 生成 Nginx 配置
	giteaConf := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
    }
}`, giteaDomain)

	syncthingConf := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    location / {
        proxy_pass http://localhost:8384;
        proxy_set_header Host $host;
    }
}`, syncthingDomain)

	os.WriteFile("/etc/nginx/sites-available/gitea", []byte(giteaConf), 0644)
	os.WriteFile("/etc/nginx/sites-available/syncthing", []byte(syncthingConf), 0644)
	os.Symlink("/etc/nginx/sites-available/gitea", "/etc/nginx/sites-enabled/")
	os.Symlink("/etc/nginx/sites-available/syncthing", "/etc/nginx/sites-enabled/")
	runCmd("systemctl", "restart", "nginx")

	// 调用 Certbot 申请证书
	runCmd("certbot", "--nginx", "-d", giteaDomain, "-d", syncthingDomain, "--non-interactive", "--agree-tos", "--email", "admin@"+giteaDomain)
	runCmd("systemctl", "restart", "nginx")
}

// 安装 frp
func setupFrp() {
	fmt.Println(Yellow + "[6/6] 安装 frp (内网穿透)..." + Reset)
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入 frp 服务端 IP：")
	frpServer, _ := reader.ReadString('\n')
	frpServer = strings.TrimSpace(frpServer)

	fmt.Print("请输入 frp 服务端端口（默认 7000）：")
	frpPort, _ := reader.ReadString('\n')
	frpPort = strings.TrimSpace(frpPort)
	if frpPort == "" {
		frpPort = "7000"
	}

	// 下载并配置 frp
	runCmd("wget", "https://github.com/fatedier/frp/releases/download/v0.52.3/frp_0.52.3_linux_amd64.tar.gz")
	runCmd("tar", "-xzf", "frp_0.52.3_linux_amd64.tar.gz")
	os.Chdir("frp_0.52.3_linux_amd64")

	frpConfig := fmt.Sprintf(`[common]
server_addr = %s
server_port = %s

[gitea]
type = tcp
local_ip = 127.0.0.1
local_port = 3000
remote_port = 7001

[syncthing]
type = tcp
local_ip = 127.0.0.1
local_port = 8384
remote_port = 7002`, frpServer, frpPort)

	os.WriteFile("frpc.ini", []byte(frpConfig), 0644)
	exec.Command("./frpc", "-c", "frpc.ini").Start()
	fmt.Println(Green + "frpc 已启动（后台运行）。如需持久化，请配置 systemd。" + Reset)
}

// 主菜单
func mainMenu() {
	checkRoot()
	fmt.Println(Green + "=== diyNAS 自动化部署工具 ===" + Reset)
	fmt.Println("1. 一键安装所有组件")
	fmt.Println("2. 仅安装 Samba (内网共享)")
	fmt.Println("3. 仅安装 Gitea (代码托管)")
	fmt.Println("4. 仅安装 Syncthing (文件同步)")
	fmt.Println("5. 仅配置 Nginx (HTTPS 代理)")
	fmt.Println("6. 仅安装 frp (内网穿透)")
	fmt.Println("7. 退出")

	var choice int
	fmt.Print("请输入选项 [1-7]：")
	fmt.Scanln(&choice)

	switch choice {
	case 1:
		installDeps()
		setupSamba()
		setupGitea()
		setupSyncthing()
		setupNginx()
		setupFrp()
	case 2:
		setupSamba()
	case 3:
		setupGitea()
	case 4:
		setupSyncthing()
	case 5:
		setupNginx()
	case 6:
		setupFrp()
	case 7:
		os.Exit(0)
	default:
		fmt.Println(Red + "无效选项！" + Reset)
		mainMenu()
	}
}

func main() {
	mainMenu()
}
