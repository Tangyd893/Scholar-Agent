// PDF Worker — RabbitMQ 消费者
//
// 职责：
//   - 消费 pdf.parse 队列中的 PDF 解析任务
//   - Mock OCR：模拟文本提取（Phase 2 阶段，后续可接入百度/Paddle OCR）
//   - 文本分块：512 token / 50 overlap
//   - Embedding 向量化
//   - Qdrant 入库
//
// Phase 2 实现；Phase 3 可替换 OCR provider。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/redis/go-redis/v9"

	"github.com/Tangyd893/Scholar-Agent/pkg/embedding"
	"github.com/Tangyd893/Scholar-Agent/pkg/qdrant"
)

const (
	collection    = "papers"
	embeddingDim  = 1536
	chunkSize     = 512 // 每块约 512 token（英文约 2000 字符）
	chunkOverlap  = 50
)

type job struct {
	JobID     string `json:"job_id"`
	FileID    string `json:"file_id"`
	SessionID string `json:"session_id"`
}

func main() {
	mqURL := os.Getenv("RABBITMQ_URL")
	if mqURL == "" {
		mqURL = "amqp://guest:guest@localhost:5672/"
	}

	// 连接 RabbitMQ
	conn, err := amqp.Dial(mqURL)
	if err != nil {
		slog.Error("pdf-worker: cannot connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		slog.Error("pdf-worker: cannot open channel", "error", err)
		os.Exit(1)
	}
	defer ch.Close()

	// 声明队列（幂等）
	q, err := ch.QueueDeclare("pdf.parse", true, false, false, false, nil)
	if err != nil {
		slog.Error("pdf-worker: cannot declare queue", "error", err)
		os.Exit(1)
	}

	// 限流：每次取 1 条
	ch.Qos(1, 0, false)

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		slog.Error("pdf-worker: cannot consume", "error", err)
		os.Exit(1)
	}

	// 初始化 Qdrant + Embedding
	// Redis 状态上报
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	redisOpts, _ := redis.ParseURL(redisURL)
	redisClient := redis.NewClient(redisOpts)

	qdrantClient := qdrant.NewClient(collection)
	embedClient, err := embedding.NewClient()
	if err != nil {
		slog.Error("pdf-worker: embedding init failed", "error", err)
		os.Exit(1)
	}

	if err := qdrantClient.EnsureCollection(context.Background(), embeddingDim); err != nil {
		slog.Error("pdf-worker: Qdrant collection init failed", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fmt.Println("pdf-worker started, waiting for jobs...")
	slog.Info("pdf-worker: ready", "queue", q.Name)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("pdf-worker shutting down...")
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}

			var j job
			if err := json.Unmarshal(msg.Body, &j); err != nil {
				slog.Error("pdf-worker: bad message", "error", err)
				msg.Nack(false, false)
				continue
			}

			slog.Info("pdf-worker: processing", "job_id", j.JobID)
			if err := processJob(ctx, j, embedClient, qdrantClient); err != nil {
				slog.Error("pdf-worker: job failed", "job_id", j.JobID, "error", err)
				redisClient.Set(ctx, fmt.Sprintf("job:%s:status", j.JobID), "failed", 24*time.Hour)
				msg.Nack(false, false)
			} else {
				slog.Info("pdf-worker: job completed", "job_id", j.JobID)
				redisClient.Set(ctx, fmt.Sprintf("job:%s:status", j.JobID), "completed", 24*time.Hour)
				msg.Ack(false)
			}
		}
	}
}

func processJob(ctx context.Context, j job, embed *embedding.Client, qd *qdrant.Client) error {
	// =====================================================================
	// 1. Mock OCR — 模拟从 PDF 提取文本
	// =====================================================================
	text := mockOCR(j.FileID)

	// =====================================================================
	// 2. 文本分块
	// =====================================================================
	chunks := splitText(text, chunkSize, chunkOverlap)
	slog.Info("pdf-worker: chunks", "job_id", j.JobID, "count", len(chunks))

	// =====================================================================
	// 3. Embedding + Qdrant 入库
	// =====================================================================
	for i, chunk := range chunks {
		vec, err := embed.Embed(ctx, chunk)
		if err != nil {
			return fmt.Errorf("embed chunk %d: %w", i, err)
		}

		point := qdrant.Point{
			ID:     i,
			Vector: vec,
			Payload: map[string]interface{}{
				"job_id":   j.JobID,
				"file_id":  j.FileID,
				"chunk_id": i,
				"content":  chunk,
			},
		}

		if err := qd.Upsert(ctx, []qdrant.Point{point}); err != nil {
			return fmt.Errorf("upsert chunk %d: %w", i, err)
		}
	}

	return nil
}

// mockOCR 生成模拟的论文文本（Phase 2 占位，Phase 3 接入真实 OCR）。
func mockOCR(fileID string) string {
	return fmt.Sprintf(`Abstract
This paper presents a novel approach to attention mechanisms in deep learning models. 
The key innovation is the introduction of multi-head attention, which allows the model 
to jointly attend to information from different representation subspaces at different 
positions. We demonstrate that this approach significantly outperforms traditional 
recurrent and convolutional architectures on a variety of sequence modeling tasks.

Introduction
Attention mechanisms have become an integral part of modern neural network architectures. 
Originally introduced in the context of neural machine translation, attention allows 
models to dynamically weight the importance of different parts of the input when 
generating each part of the output. This stands in contrast to traditional encoder-decoder 
architectures, which compress the entire input into a fixed-length vector.

The Transformer model, introduced by Vaswani et al. in 2017, relies entirely on attention 
mechanisms, dispensing with recurrence and convolutions entirely. This design allows for 
significantly more parallelization and achieves state-of-the-art results on many benchmarks.

Methodology
Our proposed architecture consists of stacked self-attention and point-wise, fully 
connected layers. The self-attention mechanism computes a weighted sum of values, where 
the weight assigned to each value is computed by a compatibility function of the query 
with the corresponding key. Multi-head attention allows the model to attend to different 
representations, improving the model's ability to capture diverse linguistic phenomena.

Experiments
We evaluate our approach on standard machine translation benchmarks including WMT 2014 
English-to-German and English-to-French. Our model achieves BLEU scores of 28.4 and 41.8 
respectively, surpassing all previously reported results. Additionally, we demonstrate 
the generalizability of the Transformer by applying it to English constituency parsing.

Conclusion
We have presented the Transformer, the first sequence transduction model based entirely 
on attention, replacing recurrent layers with multi-head self-attention. The model can 
be trained significantly faster than architectures based on recurrent or convolutional 
layers and achieves new state-of-the-art results on translation tasks.

References
[1] Bahdanau, D., Cho, K., & Bengio, Y. Neural Machine Translation by Jointly Learning 
to Align and Translate. ICLR 2015.
[2] Vaswani, A., et al. Attention Is All You Need. NeurIPS 2017.
[3] Devlin, J., et al. BERT: Pre-training of Deep Bidirectional Transformers. NAACL 2019.
`, fileID)
}

// splitText 将文本按指定大小分块，块之间有 overlap 个词的交叉。
func splitText(text string, chunkWords, overlapWords int) []string {
	words := strings.Fields(text)
	if len(words) <= chunkWords {
		return []string{text}
	}

	var chunks []string
	step := chunkWords - overlapWords
	if step <= 0 {
		step = chunkWords
	}

	for i := 0; i < len(words); i += step {
		end := i + chunkWords
		if end > len(words) {
			end = len(words)
		}
		chunk := strings.Join(words[i:end], " ")
		chunks = append(chunks, chunk)
		if end == len(words) {
			break
		}
	}

	return chunks
}
