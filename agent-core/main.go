// Agent Core — ReAct 推理引擎
//
// 职责：
//   - ReAct 主循环（Thought → Action → Observation → Answer）
//   - LLM Gateway（DeepSeek FC + MiMo/MiniMax 降级）
//   - gRPC Server（AgentCore.Run 流式接口）
//   - 会话历史管理（通过 Redis）
//
// Phase 1 验收：CLI 能走通 question → tool call → answer
package main

import "fmt"

func main() {
	fmt.Println("agent-core starting gRPC on :50051...")
	// TODO: 初始化 LLMClient、ToolServiceClient(gRPC)、Redis、启动 gRPC server
}
