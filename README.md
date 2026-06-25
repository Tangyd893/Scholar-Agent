<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-18-61DAFB?style=for-the-badge&logo=react&logoColor=black" alt="React">
  <img src="https://img.shields.io/badge/Docker-ready-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker">
  <img src="https://img.shields.io/badge/Redis-7-DC382D?style=for-the-badge&logo=redis&logoColor=white" alt="Redis">
  <img src="https://img.shields.io/badge/Status-~95%25-brightgreen?style=for-the-badge" alt="Status">
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

ScholarAgent 以 **ReAct + Function Calling** 驱动 Agent 自主规划检索与阅读步骤，通过 SSE 将 Thought / Action / Observation 完整推送到前端，后端采用 Go 微服务（Gateway、Agent Core、Tool Service、PDF Worker）独立部署。

**当前状态（2026-06-25）**

| 维度 | 状态 |
| ---- | ---- |
| 整体完成度 | **约 95%** |
| Phase 1 MVP | ✅ 完成 |
| Phase 2 增强 | ✅ 完成 |
| Phase 3 生产化 | 🔨 基本完成（缺演示视频） |
| CLI / Web / Docker 全链路 | ✅ 真实 ReAct 已贯通 |

剩余事项见 [docs/待办清单.md](docs/待办清单.md)。

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
├── ⚙️ .env.example
├── 📦 gateway/                 # HTTP 入口、SSE、会话 API
├── 📦 agent-core/              # ReAct 引擎、LLM Gateway
├── 📦 tool-service/            # 工具注册与 gRPC 执行
├── 📦 pdf-worker/              # RabbitMQ 消费者、PDF 入库
├── 📦 frontend/                # React + Vite Web UI
├── 📦 pkg/                     # 共享库（arxiv、qdrant、metrics 等）
├── 📦 proto/                   # gRPC 定义与生成代码
├── 📦 scripts/                 # DeepSeek FC 测试、CLI E2E 脚本
├── 🐳 deploy/
│   ├── docker-compose.yml      # 9 服务一键编排
│   ├── prometheus.yml
│   ├── grafana/
│   └── k8s/
└── 📁 docs/
    ├── 项目设计.md
    ├── 产品需求.md
    ├── 技术设计.md
    ├── API规范.md
    ├── 里程碑与验收.md
    ├── 动工前指引.md
    └── 待办清单.md
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
| **Node.js 18+** | ⭐ | 仅本地开发前端时需要 |
| **MiMo / MiniMax Key** | ⭐ | LLM 降级备选 |

> 💡 Agent 默认使用 `deepseek-v4-flash` + **non-thinking** 模式。复制 `.env.example` 为 `.env` 并填入密钥。

### 📦 Step 1 — 克隆仓库

```bash
git clone https://github.com/Tangyd893/Scholar-Agent.git
cd Scholar-Agent
```

验证 DeepSeek Function Calling：

```bash
go run scripts/test_deepseek_fc.go
```

### ⚙️ Step 2 — 配置

**Linux / macOS：**

```bash
cp .env.example .env
# 编辑 .env 填入 DEEPSEEK_API_KEY 等
export $(grep -v '^#' .env | xargs)
```

**Windows PowerShell：**

```powershell
Copy-Item .env.example .env
# 编辑 .env 后加载环境变量
Get-Content .env | ForEach-Object {
  if ($_ -match '^([^#][^=]+)=(.*)$') { Set-Item -Path "env:$($matches[1])" -Value $matches[2] }
}
```

| 字段 | 必填 | 说明 |
|:-----|:----:|------|
| `DEEPSEEK_API_KEY` | ✅ | DeepSeek API 密钥 |
| `REDIS_URL` | ✅ | 默认 `redis://localhost:6379` |
| `EMBEDDING_API_KEY` | ⭐ | RAG / PDF 向量化（可用 DeepSeek Key） |
| `MIMO_API_KEY` | ⭐ | MiMo 降级备选 |
| `MINIMAX_API_KEY` | ⭐ | MiniMax 降级备选 |

完整变量列表见 [.env.example](.env.example)。

### ▶️ Step 3 — 一键启动（推荐）

启动全部 9 个服务（Gateway、Agent Core、Tool Service、PDF Worker、Frontend、Redis、Qdrant、RabbitMQ、Prometheus、Grafana）：

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

访问 Web 界面：

```
http://localhost:8080
```

访问 Grafana（默认 admin / admin）：

```
http://localhost:3000
```

### ▶️ Step 4 — CLI 调试（可选）

仅启动 Redis：

```bash
docker compose -f deploy/docker-compose.yml up -d redis
```

Mock 模式（零 API 调用）：

```bash
go run ./agent-core/cmd/cli --query "帮我找 attention 论文" --mock
```

真实 DeepSeek + arXiv：

```bash
go run ./agent-core/cmd/cli --query "attention mechanism" --arxiv
```

CLI 端到端验收（3 次连续问答）：

```bash
bash scripts/e2e_cli_test.sh
```

---

## 📂 核心产出

| 产出 | 说明 | 用途 |
|:-----|------|:-----|
| 💬 **SSE 思考链** | `thought` / `action` / `observation` / `answer` 事件流 | 前端实时展示 Agent 推理过程 |
| 📚 **论文检索结果** | 标题、作者、年份、摘要、paper_id | Agent 筛选与推荐 |
| 📄 **RAG 知识库** | Qdrant `papers` collection 向量片段 | 针对已上传 PDF 的问答 |
| 📋 **BibTeX 引用** | `generate_citation` 工具输出 | 直接用于 LaTeX 写作 |
| 📊 **Prometheus 指标** | 工具调用、LLM 延迟、推理步数 | Grafana 可观测性 |

---

## ✨ 特性

### 🧠 透明 ReAct 推理

Agent 基于 Function Calling 自主决定何时搜索论文、获取摘要或查询知识库，每一步通过 SSE 完整推送。

### 📖 学术专项工具链

| 工具 | 能力 | 状态 |
|:-----|------|:----:|
| `search_papers` | arXiv 关键词检索 | ✅ |
| `get_abstract` | 按 paper_id 获取摘要 | ✅ |
| `rag_query` | Qdrant 向量检索 | ✅ |
| `generate_citation` | BibTeX 引用生成 | ✅ |
| `parse_pdf` | 异步 OCR + 分块 + 入库 | ✅ mock OCR |

### 🏗️ Go 微服务架构

Gateway、Agent Core、Tool Service、PDF Worker 独立部署，gRPC 通信，Redis 会话，RabbitMQ 异步 PDF 解析。

### 🔄 LLM 多级降级

`deepseek-v4-flash`（non-thinking）→ MiMo V2.5 Pro → MiniMax-M3；Function Calling 失败时回退 JSON Mode。

---

## 🤖 API 集成

```bash
# 创建会话
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"title": "attention 调研"}'

# 流式问答（SSE）
curl -N -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"session_id": "sess_xxx", "query": "推荐 3 篇 attention 经典论文"}'
```

完整定义见 [docs/API规范.md](docs/API规范.md)。

---

## 📋 里程碑

| Phase | 目标 | 状态 |
|:-----:|------|:----:|
| **Phase 1** | ReAct + 2 工具 + Redis + 微服务 + CLI | ✅ |
| **Phase 2** | RAG + PDF 异步 + Web UI + Docker 全栈 | ✅ |
| **Phase 3** | Prometheus + Grafana + K8s + 单元测试 | 🔨 缺演示视频 |

验收详情：[docs/里程碑与验收.md](docs/里程碑与验收.md) | 剩余待办：[docs/待办清单.md](docs/待办清单.md)

---

## 📖 文档索引

| 文档 | 说明 |
|:-----|------|
| [docs/项目设计.md](docs/项目设计.md) | 项目总览与技术选型 |
| [docs/产品需求.md](docs/产品需求.md) | PRD：用户故事、NFR |
| [docs/技术设计.md](docs/技术设计.md) | 微服务架构、gRPC、RAG |
| [docs/API规范.md](docs/API规范.md) | REST / SSE 规范 |
| [docs/里程碑与验收.md](docs/里程碑与验收.md) | 三阶段 DoD |
| [docs/动工前指引.md](docs/动工前指引.md) | 施工顺序与防走偏 |
| [docs/待办清单.md](docs/待办清单.md) | 完成度与剩余事项 |

---

## ❓ 常见问题

| 问题 | 解决方案 |
|:-----|---------|
| 如何最快体验？ | `docker compose -f deploy/docker-compose.yml up -d --build`，访问 `http://localhost:8080` |
| 没有 Docker？ | `go run ./agent-core/cmd/cli --query "测试" --mock` |
| 用什么模型？ | `deepseek-v4-flash`，Agent 循环默认 non-thinking |
| PDF OCR 是真实的吗？ | 当前为 mock OCR，真实百度 MCP / PaddleOCR 待接入 |
| 需要登录吗？ | MVP 使用匿名 `device_id` Cookie |

---

## ⏰ 部署

**Docker Compose（本地全栈）：**

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

**Kubernetes（Minikube）：**

```bash
minikube start
kubectl apply -f deploy/k8s/
```

---

## 📜 License

待定

---

<p align="center">
  <b>⭐ 觉得有用？点个 Star 支持一下！</b>
</p>
