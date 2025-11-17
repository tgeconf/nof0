# NOF0 快速启动指南

本指南将帮助你快速运行 NOF0 项目。

## 📋 前置要求

### 必需工具

1. **Node.js** (版本 18+)
   ```bash
   # 使用 Homebrew 安装 (macOS)
   brew install node
   
   # 或访问 https://nodejs.org/ 下载安装
   ```

2. **npm** (通常随 Node.js 一起安装)
   ```bash
   npm --version  # 验证安装
   ```

### 可选工具（用于后端）

3. **Go** (版本 1.22+)
   
   **方法一：使用 Homebrew 安装（推荐）**
   
   如果下载速度慢，可以使用国内镜像加速：
   
   ```bash
# 配置 Homebrew 核心镜像
git -C "$(brew --repo)" remote set-url origin https://mirrors.tuna.tsinghua.edu.cn/git/homebrew/brew.git

# 配置 Homebrew 公式镜像
git -C "$(brew --repo homebrew/core)" remote set-url origin https://mirrors.tuna.tsinghua.edu.cn/git/homebrew/homebrew-core.git

# 配置 Homebrew cask 镜像
git -C "$(brew --repo homebrew/cask)" remote set-url origin https://mirrors.tuna.tsinghua.edu.cn/git/homebrew/homebrew-cask.git

# 配置 Homebrew Bottles 镜像
export HOMEBREW_BOTTLE_DOMAIN=https://mirrors.tuna.tsinghua.edu.cn/homebrew-bottles
   
   # 2. 更新 Homebrew
   brew update
   
   # 3. 安装 Go
   brew install go
   ```
   
   **方法二：直接下载安装包（最快）**
   
   ```bash
   # 访问 https://go.dev/dl/ 下载 macOS 安装包
   # 或使用命令行下载（以 Go 1.22 为例）
   curl -L -o go.pkg https://go.dev/dl/go1.22.3.darwin-amd64.pkg
   # 然后双击 go.pkg 安装
   ```
   
   **方法三：使用代理（如果有）**
   
   ```bash
   export http_proxy=http://your_proxy:port
   export https_proxy=http://your_proxy:port
   brew install go
   ```

## 🚀 快速开始

### 方式一：仅启动前端（推荐，最简单）

前端可以独立运行，使用内置的 API 代理访问 nof1.ai 的数据。

```bash
# 1. 进入前端目录
cd web

# 2. 安装依赖
npm install

# 3. 启动开发服务器
npm run dev
```

访问 **http://localhost:3000** 即可查看前端界面。

**说明**：
- 前端会通过 Next.js API 路由代理到 nof1.ai 的 API
- 无需启动后端即可查看界面和演示数据
- 这是最快的体验方式

---

### 方式二：启动完整系统（前端 + 后端）

#### 步骤 1: 启动前端

```bash
cd web
npm install
npm run dev
```

前端将在 `http://localhost:3000` 运行。

#### 步骤 2: 配置后端环境变量（可选）

如果需要使用本地后端，需要设置环境变量：

```bash
# 进入后端目录
cd go

# 设置 LLM API Key（如果使用 LLM 功能）
export ZENMUX_API_KEY=your_api_key_here

# 设置 Hyperliquid 私钥（如果使用交易所功能）
export HYPERLIQUID_PRIVATE_KEY=your_private_key_here
```

**注意**：如果只是查看数据，可以跳过环境变量设置，后端会使用文件数据源。

#### 步骤 3: 启动后端

```bash
# 确保在 go 目录下
cd go

# 安装 Go 依赖
go mod download

# 构建并运行
go build -o nof0-api ./nof0.go
./nof0-api -f etc/nof0.yaml
```

后端将在 `http://localhost:8888` 运行。

#### 步骤 4: 配置前端连接本地后端

创建或修改 `web/.env.local` 文件：

```bash
NEXT_PUBLIC_NOF1_API_BASE_URL=http://localhost:8888
```

然后重启前端服务。

---

## 🔍 验证安装

### 检查前端

访问 http://localhost:3000，你应该看到：
- 首页展示 AI 交易竞技场
- 排行榜数据
- 账户总资产曲线
- 持仓情况
- 成交记录

### 检查后端（如果启动）

```bash
# 测试 API 端点
curl http://localhost:8888/api/crypto-prices
curl http://localhost:8888/api/leaderboard
curl http://localhost:8888/api/trades
```

---

## 🐛 常见问题

### 1. Node.js 未安装

**错误**：`command not found: node`

**解决**：
```bash
# macOS
brew install node

# 或访问 https://nodejs.org/ 下载安装
```

### 2. 端口被占用

**错误**：`Port 3000 is already in use`

**解决**：
```bash
# 查找占用端口的进程
lsof -ti:3000

# 杀死进程
kill $(lsof -ti:3000)

# 或使用其他端口
npm run dev -- -p 3001
```

### 3. 后端端口被占用

**错误**：`bind: address already in use`

**解决**：
```bash
# 查找占用 8888 端口的进程
lsof -ti:8888

# 杀死进程
kill $(lsof -ti:8888)
```

### 4. npm install 失败

**错误**：网络问题或权限问题

**解决**：
```bash
# 清除缓存
npm cache clean --force

# 使用国内镜像（可选）
npm config set registry https://registry.npmmirror.com

# 重新安装
npm install
```

### 5. Go 模块下载失败

**错误**：`go: module ... not found`

**解决**：
```bash
# 设置 Go 代理（国内用户）
go env -w GOPROXY=https://goproxy.cn,direct

# 重新下载
go mod download
```

### 6. 数据文件未找到

**错误**：后端找不到数据文件

**解决**：
- 确保 `mcp/data` 目录存在
- 检查 `go/etc/nof0.yaml` 中的 `DataPath` 配置
- 默认路径应该是 `../mcp/data`

---

## 📚 下一步

- 查看 [完整文档](https://wquguru.gitbook.io/nof0)
- 阅读 [后端 README](go/README.md) 了解后端详细配置
- 查看 [前端文档](web/docs/) 了解前端开发规范

---

## 💡 提示

1. **最快体验**：只启动前端即可，它会自动代理到 nof1.ai 的 API
2. **本地开发**：启动后端后，前端可以通过环境变量切换到本地 API
3. **数据源**：后端支持文件数据源（JSON）和数据库模式（Postgres+Redis）
4. **开发模式**：后端默认使用 `test` 环境，会使用低成本的 LLM 模型

---

**祝你使用愉快！** 🎉
