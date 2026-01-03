# 端口配置指南

## 🎯 配置方式

CLI Gateway 支持多种方式配置端口，优先级从高到低：

1. **环境变量** (最高优先级)
2. **配置文件**
3. **默认值** (8080)

## 📋 配置方法

### 方式 1: 环境变量

最简单的方式，适合临时修改或容器部署。

```bash
# 设置端口
export PORT=3000
./start.sh

# 或者一行命令
PORT=3000 ./start.sh

# 同时设置主机和端口
HOST=127.0.0.1 PORT=9000 ./start.sh
```

### 方式 2: 启动脚本参数

使用 `start.sh` 的命令行参数：

```bash
# 使用 -p 或 --port 参数
./start.sh -p 3000
./start.sh --port 3000

# 查看帮助
./start.sh --help
```

### 方式 3: 配置文件

编辑 `configs/configs.json`（或 `configs.json`）：

```json
{
  "server": {
    "port": 3000,
    "host": "0.0.0.0"
  },
  "profiles": {
    ...
  }
}
```

配置说明：
- `port`: 端口号，默认 8080
- `host`: 监听地址
  - `0.0.0.0` - 监听所有网络接口（默认）
  - `127.0.0.1` - 仅本地访问
  - `::` - IPv6 所有接口

## 🔧 使用示例

### 示例 1: 开发环境（端口 3000）

```bash
./start.sh -p 3000
```

访问：`http://localhost:3000`

### 示例 2: 生产环境（端口 80）

```bash
# 需要 root 权限
sudo PORT=80 ./claude-cli-gateway
```

访问：`http://your-domain.com`

### 示例 3: 仅本地访问（端口 8080）

编辑 `configs/configs.json`:
```json
{
  "server": {
    "port": 8080,
    "host": "127.0.0.1"
  }
}
```

只能通过 `http://localhost:8080` 访问，外部无法访问。

### 示例 4: Docker 容器

```dockerfile
# Dockerfile
FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o gateway ./cmd/server
EXPOSE 8080
CMD ["./gateway"]
```

```bash
# 运行容器，映射到主机的 3000 端口
docker run -p 3000:8080 -e PORT=8080 your-image
```

### 示例 5: Systemd 服务

创建 `/etc/systemd/system/cli-gateway.service`:
```ini
[Unit]
Description=CLI Gateway Service
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/cli-agent
Environment="PORT=8080"
Environment="HOST=0.0.0.0"
ExecStart=/path/to/cli-agent/claude-cli-gateway
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable cli-gateway
sudo systemctl start cli-gateway
```

## 🌐 反向代理配置

### Nginx

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Caddy

```
your-domain.com {
    reverse_proxy localhost:8080
}
```

### Apache

```apache
<VirtualHost *:80>
    ServerName your-domain.com
    
    ProxyPreserveHost On
    ProxyPass / http://localhost:8080/
    ProxyPassReverse / http://localhost:8080/
</VirtualHost>
```

## 🔍 验证配置

### 检查端口是否被占用

```bash
# Linux/macOS
lsof -i :8080

# 或者
netstat -an | grep 8080
```

### 测试服务

```bash
# 启动服务
./start.sh -p 3000

# 在另一个终端测试
curl http://localhost:3000/release-notes
```

## ⚠️ 注意事项

1. **端口范围**: 
   - 1-1023: 需要 root 权限
   - 1024-65535: 普通用户可用

2. **防火墙**: 
   - 确保防火墙允许访问该端口
   - 云服务器需要在安全组中开放端口

3. **端口冲突**: 
   - 确保端口未被其他程序占用
   - 使用 `lsof` 或 `netstat` 检查

4. **IPv6**: 
   - 如果使用 IPv6，设置 `host: "::"`
   - 确保系统支持 IPv6

## 🐛 故障排除

### 问题 1: 端口被占用

```
Error: listen tcp :8080: bind: address already in use
```

**解决**:
```bash
# 查找占用端口的进程
lsof -i :8080

# 杀死进程
kill -9 <PID>

# 或者使用其他端口
./start.sh -p 8081
```

### 问题 2: 权限不足

```
Error: listen tcp :80: bind: permission denied
```

**解决**:
```bash
# 使用 sudo
sudo PORT=80 ./claude-cli-gateway

# 或者使用高端口
./start.sh -p 8080
```

### 问题 3: 无法从外部访问

**检查**:
1. 确认 `host` 设置为 `0.0.0.0`
2. 检查防火墙规则
3. 检查云服务器安全组

```bash
# 测试本地访问
curl http://localhost:8080/release-notes

# 测试外部访问
curl http://your-ip:8080/release-notes
```

## 📚 相关文档

- [配置文件说明](../configs/configs.json)
- [启动脚本](../start.sh)
- [部署指南](./DEPLOYMENT.md)
