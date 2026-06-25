// Gateway — ScholarAgent HTTP 入口服务
//
// 职责：
//   - REST API（会话管理）
//   - SSE 流式推送（POST /api/v1/chat/stream）
//   - PDF 上传 + 解析进度查询
//   - gRPC 客户端 → Agent Core / Tool Service
//   - 静态资源托管（前端 dist/）
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Tangyd893/Scholar-Agent/pkg/metrics"
	"github.com/Tangyd893/Scholar-Agent/pkg/session"
	agentpb "github.com/Tangyd893/Scholar-Agent/proto/gen/agent"
	toolpb "github.com/Tangyd893/Scholar-Agent/proto/gen/tool"
)

// jobRecord 记录 PDF 解析任务状态。
type jobRecord struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"` // pending | processing | completed | failed
	Progress  int    `json:"progress"`
	Error     string `json:"error,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

// jobTracker 是内存中的 job 状态存储。
type jobTracker struct {
	mu   sync.RWMutex
	jobs map[string]*jobRecord
}

func newJobTracker() *jobTracker {
	return &jobTracker{jobs: make(map[string]*jobRecord)}
}

func (t *jobTracker) create(jobID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.jobs[jobID] = &jobRecord{
		JobID:     jobID,
		Status:    "pending",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func (t *jobTracker) update(jobID, status string, progress int, errMsg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if j, ok := t.jobs[jobID]; ok {
		j.Status = status
		j.Progress = progress
		j.Error = errMsg
		j.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
}

func (t *jobTracker) get(jobID string) *jobRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.jobs[jobID]
}

func main() {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	agentAddr := os.Getenv("AGENT_CORE_GRPC_ADDR")
	if agentAddr == "" {
		agentAddr = "localhost:50051"
	}
	toolAddr := os.Getenv("TOOL_SERVICE_GRPC_ADDR")
	if toolAddr == "" {
		toolAddr = "localhost:50052"
	}

	// gRPC 连接
	agentConn, err := grpc.NewClient(agentAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("gateway: agent-core connect failed", "error", err)
		os.Exit(1)
	}
	defer agentConn.Close()
	agentClient := agentpb.NewAgentCoreClient(agentConn)

	toolConn, err := grpc.NewClient(toolAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Warn("gateway: tool-service connect failed (PDF upload disabled)", "error", err)
	}
	var toolClient toolpb.ToolServiceClient
	if toolConn != nil {
		defer toolConn.Close()
		toolClient = toolpb.NewToolServiceClient(toolConn)
	}

	// Redis 会话存储
	var sessionStore *session.Store
	rs, redisErr := session.NewStore()
	if redisErr != nil {
		slog.Warn("gateway: Redis unavailable, sessions won't persist", "error", redisErr)
	} else {
		sessionStore = rs
		slog.Info("gateway: using Redis session store")
	}

	jobs := newJobTracker()
	mux := http.NewServeMux()

	// =====================================================================
	// GET /metrics — Prometheus
	// =====================================================================
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.Handler().ServeHTTP(w, r)
	})

	// =====================================================================
	// GET /health
	// =====================================================================
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// =====================================================================
	// POST /api/v1/chat/stream — SSE
	// =====================================================================
	mux.HandleFunc("POST /api/v1/chat/stream", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string `json:"session_id"`
			Query     string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}
		if req.Query == "" {
			http.Error(w, `{"error":"query required"}`, http.StatusBadRequest)
			return
		}
		if req.SessionID == "" {
			req.SessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, _ := w.(http.Flusher)
		stream, err := agentClient.Run(r.Context(), &agentpb.RunRequest{
			SessionId: req.SessionID, Query: req.Query,
		})
		if err != nil {
			writeSSE(w, "error", fmt.Sprintf(`{"type":"error","content":"%v"}`, err))
			if flusher != nil {
				flusher.Flush()
			}
			return
		}
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				writeSSE(w, "error", fmt.Sprintf(`{"type":"error","content":"%v"}`, err))
				if flusher != nil {
					flusher.Flush()
				}
				return
			}
			data, _ := json.Marshal(map[string]interface{}{
				"type": event.Type, "content": event.Content,
				"step": event.Step, "tool_args": event.ToolArgsJson,
				"timestamp": event.Timestamp.AsTime().Format(time.RFC3339),
			})
			writeSSE(w, "", string(data))
			if flusher != nil {
				flusher.Flush()
			}
		}
	})

	// =====================================================================
	// Sessions (Redis-backed)
	// =====================================================================
	mux.HandleFunc("POST /api/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Title string `json:"title"` }
		json.NewDecoder(r.Body).Decode(&req)
		deviceID := getDeviceID(r)
		var sid string
		if sessionStore != nil {
			sid, _ = sessionStore.Create(deviceID, req.Title)
		}
		if sid == "" {
			sid = fmt.Sprintf("sess_%d", time.Now().UnixNano())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session_id": sid, "title": req.Title,
			"created_at": time.Now().UTC().Format(time.RFC3339),
		})
	})
	mux.HandleFunc("GET /api/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"sessions": []interface{}{}})
	})
	mux.HandleFunc("GET /api/v1/sessions/{id}/messages", func(w http.ResponseWriter, r *http.Request) {
		sid := r.PathValue("id")
		if sessionStore == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"messages": []interface{}{}})
			return
		}
		msgs, err := sessionStore.GetMessages(sid, 50)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"messages": []interface{}{}})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"messages": msgs})
	})

	// =====================================================================
	// POST /api/v1/papers/upload — PDF 上传
	// =====================================================================
	mux.HandleFunc("POST /api/v1/papers/upload", func(w http.ResponseWriter, r *http.Request) {
		if toolClient == nil {
			http.Error(w, `{"error":"PDF upload not available"}`, http.StatusServiceUnavailable)
			return
		}

		// 限制 10MB
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, `{"error":"file too large or invalid"}`, http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, `{"error":"file field required"}`, http.StatusBadRequest)
			return
		}
		defer file.Close()

		sessionID := r.FormValue("session_id")

		// 保存到临时目录
		uploadDir := filepath.Join(os.TempDir(), "scholar-uploads")
		os.MkdirAll(uploadDir, 0755)
		fileID := fmt.Sprintf("file_%d_%s", time.Now().UnixNano(), header.Filename)
		dstPath := filepath.Join(uploadDir, fileID)

		dst, err := os.Create(dstPath)
		if err != nil {
			http.Error(w, `{"error":"save failed"}`, http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		io.Copy(dst, file)

		// 调用 IngestPDF gRPC
		jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())
		jobs.create(jobID)

		go func() {
			jobs.update(jobID, "processing", 30, "")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			resp, err := toolClient.IngestPDF(ctx, &toolpb.IngestRequest{
				FileId: fileID, SessionId: sessionID,
			})
			if err != nil {
				jobs.update(jobID, "failed", 0, err.Error())
				return
			}
			if resp.JobId != "" {
				jobID = resp.JobId
			}
			jobs.update(jobID, "completed", 100, "")
			slog.Info("gateway: PDF ingest done", "job_id", jobID)
		}()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id": jobID, "status": "pending",
			"created_at": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// =====================================================================
	// GET /api/v1/jobs/{id} — 查询 PDF 解析进度
	// =====================================================================
	mux.HandleFunc("GET /api/v1/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("id")
		job := jobs.get(jobID)
		if job == nil {
			http.Error(w, `{"error":"job not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
	})

	// =====================================================================
	// 静态文件（前端 dist/）
	// =====================================================================
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "frontend/dist"
	}
	if _, err := os.Stat(staticDir); err == nil {
		fs := http.FileServer(http.Dir(staticDir))
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			// 不拦截 /api /health /metrics
			p := r.URL.Path
			if strings.HasPrefix(p, "/api") || p == "/health" || p == "/metrics" {
				http.NotFound(w, r)
				return
			}
			fs.ServeHTTP(w, r)
		})
		slog.Info("gateway: serving static files", "dir", staticDir)
	}

	// =====================================================================
	// 启动
	// =====================================================================
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	addr := ":" + port
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		fmt.Printf("gateway listening on %s\n", addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("gateway failed", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("gateway shutting down...")
	srv.Shutdown(context.Background())
}

func getDeviceID(r *http.Request) string {
	if id := r.Header.Get("X-Device-ID"); id != "" {
		return id
	}
	if c, err := r.Cookie("device_id"); err == nil {
		return c.Value
	}
	return "anonymous"
}

func writeSSE(w io.Writer, event, data string) {
	if event != "" {
		fmt.Fprintf(w, "event: %s\n", event)
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
}
