# ScholarAgent API 规范

> 版本：v1.1 | 更新日期：2026-06-25 | Base URL: `http://localhost:8080`

---

## 一、通用约定

### 1.1 协议与格式

- REST 接口使用 JSON 请求体，`Content-Type: application/json`
- 流式聊天使用 `POST` + `text/event-stream`（**不使用 GET 传 query**，避免 URL 长度与敏感信息泄露）
- 时间戳使用 ISO 8601 UTC 格式

### 1.2 身份标识（MVP）

| 方式 | 说明 |
| ---- | ---- |
| `X-Device-ID` Header | 客户端生成的 UUID，首次访问时由 Gateway 签发并写入 Cookie |
| Cookie `device_id` | Web 端自动携带，与 Header 二选一，Gateway 优先读 Header |

会话与用户通过 `device_id` 关联，无需登录。

### 1.3 统一错误响应

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "session_id is required",
    "details": {}
  }
}
```

| HTTP 状态码 | 含义 | 典型 `code` |
| ----------- | ---- | ------------- |
| 400 | 请求参数错误 | `INVALID_REQUEST` |
| 404 | 资源不存在 | `NOT_FOUND` |
| 429 | 限流 | `RATE_LIMITED` |
| 500 | 服务内部错误 | `INTERNAL_ERROR` |
| 503 | LLM 降级中 / 服务暂不可用 | `SERVICE_UNAVAILABLE` |

---

## 二、会话 API

### POST `/api/v1/sessions`

创建新会话。

**请求体**（可选）：

```json
{
  "title": "attention 机制调研"
}
```

**响应** `201 Created`：

```json
{
  "session_id": "sess_abc123",
  "title": "attention 机制调研",
  "created_at": "2026-06-25T10:00:00Z"
}
```

### GET `/api/v1/sessions`

列出当前 `device_id` 下的会话，按 `updated_at` 降序。

**响应** `200 OK`：

```json
{
  "sessions": [
    {
      "session_id": "sess_abc123",
      "title": "attention 机制调研",
      "created_at": "2026-06-25T10:00:00Z",
      "updated_at": "2026-06-25T10:30:00Z"
    }
  ]
}
```

### GET `/api/v1/sessions/{session_id}/messages`

获取会话历史消息与 Agent 推理链。

**响应** `200 OK`：

```json
{
  "messages": [
    {
      "role": "user",
      "content": "帮我找 attention 相关的经典论文",
      "timestamp": "2026-06-25T10:00:00Z"
    },
    {
      "role": "assistant",
      "content": "推荐以下 3 篇...",
      "steps": [
        {"type": "thought", "content": "需要先搜索...", "step": 1},
        {"type": "action", "content": "search_papers", "step": 1},
        {"type": "observation", "content": "找到 15 篇...", "step": 1}
      ],
      "timestamp": "2026-06-25T10:00:15Z"
    }
  ]
}
```

---

## 三、聊天 API（SSE）

### POST `/api/v1/chat/stream`

发起一次学术问答，响应为 Server-Sent Events 流。

**请求体**：

```json
{
  "session_id": "sess_abc123",
  "query": "帮我找 attention 机制相关的经典论文"
}
```

**响应头**：

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

**SSE 数据格式**：

每条事件为一个 JSON 对象，放在 `data:` 行后：

```
data: {"type":"thought","content":"需要先搜索相关论文","step":1,"timestamp":"2026-06-25T10:00:01Z"}

data: {"type":"action","content":"search_papers","step":1,"tool_args":{"query":"attention mechanism"},"timestamp":"2026-06-25T10:00:02Z"}

data: {"type":"observation","content":"找到 15 篇相关论文...","step":1,"timestamp":"2026-06-25T10:00:05Z"}

data: {"type":"answer","content":"推荐以下 3 篇经典论文...","step":2,"timestamp":"2026-06-25T10:00:12Z"}
```

### 3.1 事件类型

| type | 说明 | 必填字段 |
| ---- | ---- | -------- |
| `thought` | Agent 推理思考 | `content`, `step`, `timestamp` |
| `action` | 工具调用 | `content`（工具名）, `tool_args`, `step`, `timestamp` |
| `observation` | 工具执行结果 | `content`, `step`, `timestamp` |
| `answer` | 最终回答（流结束） | `content`, `step`, `timestamp` |
| `error` | 错误（含 max steps 超限） | `content`, `timestamp` |
| `job_complete` | 异步 PDF 解析完成（可选推送） | `content`（job_id）, `timestamp` |

### 3.2 前端接入示例

```typescript
async function streamChat(sessionId: string, query: string) {
  const response = await fetch('/api/v1/chat/stream', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ session_id: sessionId, query }),
  });

  const reader = response.body!.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    const lines = decoder.decode(value).split('\n');
    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = JSON.parse(line.slice(6));
        handleEvent(data);
      }
    }
  }
}
```

---

## 四、PDF 与异步任务 API

### POST `/api/v1/papers/upload`

上传 PDF 文件，触发异步解析与 RAG 入库。

**请求**：`multipart/form-data`

| 字段 | 类型 | 说明 |
| ---- | ---- | ---- |
| `file` | file | PDF 文件，最大 10MB（Phase 2） |
| `session_id` | string | 关联会话（可选） |

**响应** `202 Accepted`：

```json
{
  "job_id": "job_xyz789",
  "status": "pending",
  "created_at": "2026-06-25T10:05:00Z"
}
```

### GET `/api/v1/jobs/{job_id}`

查询 PDF 解析任务进度。

**响应** `200 OK`：

```json
{
  "job_id": "job_xyz789",
  "status": "processing",
  "progress": 60,
  "paper_id": "paper_def456",
  "error": null,
  "updated_at": "2026-06-25T10:06:00Z"
}
```

| status | 说明 |
| ------ | ---- |
| `pending` | 已入队，等待 Worker |
| `processing` | OCR / 分块 / Embedding 进行中 |
| `completed` | 已写入 Qdrant，可 RAG 查询 |
| `failed` | 解析失败，见 `error` 字段 |

---

## 五、健康检查与指标

### GET `/health`

Gateway 健康检查，返回各下游服务状态。

```json
{
  "status": "ok",
  "services": {
    "agent-core": "ok",
    "tool-service": "ok",
    "redis": "ok"
  }
}
```

### GET `/metrics`

Prometheus 指标端点（各服务独立端口，Gateway 默认 `:9090`）。

核心指标：

| 指标名 | 类型 | 标签 |
| ------ | ---- | ---- |
| `agent_tool_calls_total` | Counter | `tool_name`, `status` |
| `agent_steps_per_query` | Histogram | — |
| `llm_request_duration_seconds` | Histogram | `provider`, `model` |

---

## 六、内部 gRPC 接口（服务间）

Gateway 不直接暴露 gRPC；以下为 Agent Core ↔ Tool Service 约定，详见 [技术设计.md](./技术设计.md)。

| 服务 | RPC | 说明 |
| ---- | --- | ---- |
| AgentCore | `Run(RunRequest) returns (stream StepEvent)` | ReAct 推理主循环 |
| ToolService | `Execute(ExecuteRequest) returns (ExecuteResponse)` | 同步工具调用 |
| ToolService | `IngestPDF(IngestRequest) returns (IngestResponse)` | 提交 PDF 解析任务，返回 job_id |

---

*关联文档：[产品需求.md](./产品需求.md) | [技术设计.md](./技术设计.md) | [动工前指引.md](./动工前指引.md)*
