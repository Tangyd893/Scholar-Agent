package tool

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Tangyd893/Scholar-Agent/proto/gen/tool"
)

// GrpcRegistry 通过 gRPC 调用远程 tool-service 的工具注册表。
type GrpcRegistry struct {
	mu     sync.RWMutex
	client pb.ToolServiceClient
	conn   *grpc.ClientConn
	tools  map[string]Tool
}

// NewGrpcRegistry 创建 gRPC 工具注册表。
func NewGrpcRegistry(addr string) (*GrpcRegistry, error) {
	if addr == "" {
		addr = os.Getenv("TOOL_SERVICE_GRPC_ADDR")
		if addr == "" {
			addr = "localhost:50052"
		}
	}

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", addr, err)
	}

	return &GrpcRegistry{
		client: pb.NewToolServiceClient(conn),
		conn:   conn,
		tools:  make(map[string]Tool),
	}, nil
}

func (r *GrpcRegistry) Close() error { return r.conn.Close() }

func (r *GrpcRegistry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

func (r *GrpcRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *GrpcRegistry) List() []ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Schema:      t.Schema(),
		})
	}
	return defs
}

func (r *GrpcRegistry) Execute(ctx context.Context, name, args string) (string, error) {
	slog.Info("grpc: executing tool", "tool", name)
	resp, err := r.client.Execute(ctx, &pb.ExecuteRequest{
		ToolName:      name,
		ArgumentsJson: args,
	})
	if err != nil {
		return "", fmt.Errorf("grpc execute %s: %w", name, err)
	}
	if resp.Error != "" {
		return "", fmt.Errorf("tool %s: %s", name, resp.Error)
	}
	return resp.Result, nil
}

// RegisterMeta 注册工具的本地元数据（Schema 等），实际执行走 gRPC。
func (r *GrpcRegistry) RegisterMeta(name, desc string, schema map[string]interface{}) {
	r.Register(&toolMeta{name: name, description: desc, schema: schema})
}

type toolMeta struct {
	name, description string
	schema            map[string]interface{}
}

func (m *toolMeta) Name() string                           { return m.name }
func (m *toolMeta) Description() string                    { return m.description }
func (m *toolMeta) Schema() map[string]interface{}          { return m.schema }
func (m *toolMeta) Execute(ctx context.Context, args string) (string, error) {
	panic("toolMeta.Execute should never be called directly")
}
