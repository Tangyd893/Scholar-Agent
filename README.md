<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-18-61DAFB?style=for-the-badge&logo=react&logoColor=black" alt="React">
  <img src="https://img.shields.io/badge/Docker-ready-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker">
  <img src="https://img.shields.io/badge/Redis-7-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis">
  <img src="https://img.shields.io/badge/Status-Design_Phase-yellow?style=for-the-badge" alt="Status">
</p>

<h1 align="center">🎓 ScholarAgent</h1>
<h3 align="center">基于 ReAct 的智能学术研究辅助 Agent</h3>
<p align="center">透明可观测的多步推理，端到端完成文献检索、PDF 问答与引用生成</p>
<p align="center">
  <b>检索 → 推理 → 工具调用 → 回答 → 引用</b>
</p>

<p align="center">
  <a href="https://github.com/Tangyd893/Scholar-Agent"><img src="https://img.shields.io/github/stars/Tangyd893/Scholar-Agent?style=flat-square&logo=github" alt="Stars"></a>
  <a href="https://github.com/Tangyd893/Scholar-Agent"><img src="https://img.shields.io/github/forks/Tangyd893/Scholar-Agent?style=flat-square&logo=github" alt="Forks"></a>
  <a href="https://github.com/Tangyd893/Scholar-Agent/issues"><img src="https://img.shields.io/github/issues/Tangyd893/Scholar-Agent?style=flat-square&logo=github" alt="Issues"></a>
</p>

---

## 💡 核心理念

> **让学术调研从「多平台切换」变成「一次对话、全程可观测」。**

研究者在文献调研中常面临三类痛点：检索分散在 arXiv / Semantic Scholar 等平台、PDF 难以快速消化、引用格式需手工整理。ScholarAgent 以 **ReAct + Function Calling** 驱动 Agent 自主规划检索与阅读步骤，通过 SSE 将 Thought / Action / Observation 完整推送到前端，同时以 Go 微服务架构保证可部署、可扩展。

**当前状态**：设计文档 v1.1 已完成，代码实现尚未开始。动工前请先阅读 [docs/动工前指引.md](docs/动工前指引.md)。

---

## 🏗️ 工作流

```
用户提问              PDF 上传
    │                    │
    ▼                    ▼
 Gateway (Gin)      POST /papers/upload
    │ gRPC               │
    ▼                    ▼
 Agent Core          RabbitMQ 队列
 ReAct + LLM              │
    │ gRPC                ▼
    ▼                 PDF Worker
 Tool Service        OCR + 分块 + Embedding
    │                    │
    ├─► arXiv API        ▼
    ├─► Qdrant RAG    Qdrant 入库
    └─► MCP OCR
             │
             ▼
    SSE 流式推送（thought → action → observation → answer）
```

---

## 📁 项目结构

```
Scholar-Agent/
├── 📖 README.md
├── 🚫 .gitignore
├── 📦 gateway/                 # HTTP 入口、SSE、会话 API
├── 📦 agent-core/              # ReAct 引擎、LLM Gateway
├── 📦 tool-service/            # 工具注册与 gRPC 执行
├── 📦 pdf-worker/              # RabbitMQ 消费者（Phase 2）
├── 📦 frontend/                # React + Vite（Phase 2）
├── 📦 pkg/                     # 共享类型（StepEvent、Paper 等）
├── 📦 proto/                   # gRPC 定义
├── 📦 scripts/                 # 验证脚本（DeepSeek FC 等）
├── 🐳 deploy/
│   ├── docker-compose.yml
│   └── k8s/
└── 📁 docs/
    ├── 项目设计.md             # 总览与文档索引
    ├── 产品需求.md             # PRD
    ├── 技术设计.md             # 架构与 ADR
    ├── API规范.md              # REST / SSE
    ├── 里程碑与验收.md         # 三阶段 DoD
    └── 动工前指引.md           # 施工顺序与防走偏
```

---

## 🚀 快速开始

### ✅ 前置条件

| 依赖 | 必需 | 获取方式 |
|:-----|:----:|---------|
| **Go 1.22+** | ✅ | [go.dev/dl](https://go.dev/dl/) |
| **Docker** | ✅ | [docker.com](https://www.docker.com/) |
| **DeepSeek API Key** | ✅ | [platform.deepseek.com](https://platform.deepseek.com/) |
| **Git** | ✅ | [git-scm.com](https://git-scm.com/) |
| **Node.js 18+** | ⭐ | 仅 Phase 2 前端开发时需要 |
| **MiMo / MiniMax Key** | ⭐ | LLM 降级备选（Phase 1 末尾可选） |

> 💡 **还没有 DeepSeek Key？** 注册 [DeepSeek 开放平台](https://platform.deepseek.com/)，创建 API Key 后填入 `.env`。Agent 默认使用 `deepseek-v4-flash` + non-thinking 模式。

### 📦 Step 1 — 克隆仓库

```bash
git clone https://github.com/Tangyd893/Scholar-Agent.git
cd Scholar-Agent
```

验证 DeepSeek Function Calling（动工前推荐，不依赖微服务）：

```bash
go run scripts/test_deepseek_fc.go
```

### ⚙️ Step 2 — 配置

**Linux / macOS：**

```bash
cp .env.example .env
export $(grep -v '^#' .env | xargs)
```

**Windows PowerShell：**

```powershell
Copy-Item .env.example .env
# 编辑 .env，填入密钥后：
Get-Content .env | ForEach-Object {
  if ($_ -match '^([^#][^=]+)=(.*)$') { Set-Item -Path "env:$($matches[1])" -Value $matches[2] }
}
```

> 🤖 **让 AI 帮你生成配置：**
> ```
> 帮我生成 ScholarAgent 的 .env.example，包含 DEEPSEEK_API_KEY、REDIS_URL、
> AGENT_CORE_GRPC_ADDR、TOOL_SERVICE_GRPC_ADDR，并注明各字段含义。
> ```

| 字段 | 必填 | 说明 |
|:-----|:----:|------|
| `DEEPSEEK_API_KEY` | ✅ | DeepSeek API 密钥 |
| `REDIS_URL` | ✅ | 默认 `redis://localhost:6379` |
| `AGENT_CORE_GRPC_ADDR` | ✅ | 默认 `localhost:50051` |
| `TOOL_SERVICE_GRPC_ADDR` | ✅ | 默认 `localhost:50052` |
| `MIMO_API_KEY` | ⭐ | MiMo 降级备选 |
| `MINIMAX_API_KEY` | ⭐ | MiniMax 降级备选 |
| `AGENT_MAX_STEPS` | ❌ | ReAct 最大步数，默认 `5` |
| `SESSION_TTL_DAYS` | ❌ | 会话 TTL，默认 `7` |

### ▶️ Step 3 — 运行

> ⚠️ **代码尚未实现**，以下为 Phase 1 完成后的预期启动方式。

启动基础设施（Redis）：

```bash
docker compose -f deploy/docker-compose.yml up -d redis
```

启动全部微服务（Phase 1 完成后）：

```bash
docker compose -f deploy/docker-compose.yml up -d
```

CLI 问答（Phase 1 验收方式）：

```bash
go run ./agent-core/cmd/cli --query "帮我找 attention 机制相关的经典论文"
```

访问 Gateway 健康检查：

```bash
curl http://localhost:8080/health
```

访问 Web 界面（Phase 2 完成后）：

```
http://localhost:8080
```

---

## 📂 核心产出

| 产出 | 说明 | 用途 |
|:-----|------|:-----|
| 💬 **SSE 思考链** | `thought` / `action` / `observation` / `answer` 事件流 | 前端实时展示 Agent 推理过程 |
| 📚 **论文检索结果** | 标题、作者、年份、摘要、paper_id | Agent 筛选与推荐 |
| 📄 **RAG 知识库** | Qdrant `papers` collection 向量片段 | 针对已上传 PDF 的问答 |
| 📋 **BibTeX 引用** | `generate_citation` 工具输出 | 直接用于 LaTeX 写作 |
| 📊 **Prometheus 指标** | 工具调用、LLM 延迟、推理步数 | Grafana 可观测性（Phase 3） |

---

## ✨ 特性

### 🧠 透明 ReAct 推理

Agent 基于 Function Calling 自主决定何时搜索论文、获取摘要或查询知识库。每一步推理通过 SSE 推送，用户可完整看到「想了什么 → 调了什么工具 → 得到了什么」。

### 📖 学术专项工具链

| 工具 | 能力 |
|:-----|------|
| `search_papers` | arXiv 关键词检索（Semantic Scholar 备选） |
| `get_abstract` | 按 paper_id 获取摘要 |
| `rag_query` | Qdrant 向量检索已入库 PDF |
| `generate_citation` | 生成 BibTeX 引用 |
| `parse_pdf` | 异步 OCR + 分块 + 入库 |

### 🏗️ Go 微服务架构

Gateway、Agent Core、Tool Service、PDF Worker 独立部署，gRPC 服务间通信，Redis 会话持久化，RabbitMQ 处理耗时 PDF 解析。

### 🔄 LLM 多级降级

主力 `deepseek-v4-flash`（non-thinking）→ MiMo V2.5 Pro → MiniMax-M3；Function Calling 失败时回退 JSON Mode。

---

## 🤖 Agent 集成说明

ScholarAgent 对外暴露标准 HTTP / SSE API，可被其他 Agent 或自动化流程调用：

**创建会话并提问：**

```bash
# 1. 创建会话
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"title": "attention 调研"}'

# 2. 流式问答（SSE）
curl -N -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"session_id": "sess_xxx", "query": "推荐 3 篇 attention 经典论文"}'
```

完整 API 定义见 [docs/API规范.md](docs/API规范.md)。

---

## 📋 里程碑

| Phase | 周期 | 目标 | 状态 |
|:-----:|:----:|------|:----:|
| **Phase 1** | 1 周 | ReAct + 2 工具 + Redis + 微服务骨架 + CLI | 未开始 |
| **Phase 2** | 1 周 | RAG + PDF 异步解析 + Web UI + Docker | 未开始 |
| **Phase 3** | 1 周 | Prometheus + Grafana + K8s + 测试 + 演示 | 未开始 |

验收标准详见 [docs/里程碑与验收.md](docs/里程碑与验收.md)。

---

## 📖 文档索引

| 文档 | 说明 |
|:-----|------|
| [docs/项目设计.md](docs/项目设计.md) | 项目总览与技术选型 |
| [docs/产品需求.md](docs/产品需求.md) | PRD：用户故事、NFR、范围边界 |
| [docs/技术设计.md](docs/技术设计.md) | 微服务架构、gRPC、RAG、ADR |
| [docs/API规范.md](docs/API规范.md) | REST / SSE 端点与事件 Schema |
| [docs/里程碑与验收.md](docs/里程碑与验收.md) | 三阶段 Definition of Done |
| [docs/动工前指引.md](docs/动工前指引.md) | 开工顺序与防走偏原则 |

---

## ❓ 常见问题

| 问题 | 解决方案 |
|:-----|---------|
| 项目能跑吗？ | 当前仅设计文档阶段，请按 [动工前指引](docs/动工前指引.md) 从 Phase 1 脚手架开始 |
| 用什么模型？ | 主力 `deepseek-v4-flash`，Agent 循环默认 **non-thinking** 模式 |
| Phase 1 需要前端吗？ | 不需要，CLI 问答即验收标准 |
| `deepseek-chat` 还能用吗？ | legacy ID，2026-07-24 退役，请迁移至 `deepseek-v4-flash` |
| 微服务是否过重？ | 已选定全微服务；Phase 1 先竖切 agent-core 单链路，再拆 gRPC |
| PDF 数据存在哪？ | 向量默认存本地 Qdrant；云端 OCR（百度 MCP）为可选路径 |
| 需要登录吗？ | MVP 使用匿名 `device_id` Cookie，无注册登录 |

---

## ⏰ 部署参考（Phase 2+）

**本地开发 — Docker Compose：**

```bash
docker compose -f deploy/docker-compose.yml up -d
```

**生产演示 — Kubernetes（Minikube）：**

```bash
minikube start
kubectl apply -f deploy/k8s/
```

**CI/CD — GitHub Actions（规划中）：**

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go build ./...
      - run: go test ./...
```

---

## 📖 环境变量参考

### Agent Core

```
DEEPSEEK_API_KEY          DeepSeek API 密钥
DEEPSEEK_MODEL            模型 ID，默认 deepseek-v4-flash
DEEPSEEK_THINKING         是否启用 thinking，默认 disabled
AGENT_MAX_STEPS           ReAct 最大步数，默认 5
TOOL_SERVICE_GRPC_ADDR    Tool Service 地址
REDIS_URL                 Redis 连接串
```

### Gateway

```
HTTP_PORT                 HTTP 端口，默认 8080
AGENT_CORE_GRPC_ADDR      Agent Core 地址
REDIS_URL                 Redis 连接串
```

### Tool Service

```
TOOL_SERVICE_GRPC_PORT    gRPC 端口，默认 50052
ARXIV_RATE_LIMIT_MS       arXiv 请求间隔，默认 3000
QDRANT_URL                Qdrant 地址（Phase 2）
RABBITMQ_URL              RabbitMQ 地址（Phase 2）
```

---

## 📜 License

待定

---

<p align="center">
  <b>⭐ 觉得有用？点个 Star 支持一下！</b>
</p>
