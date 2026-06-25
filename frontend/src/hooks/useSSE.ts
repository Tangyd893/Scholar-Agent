import { useState, useCallback, useRef } from "react";

export interface StepEvent {
  type: "thought" | "action" | "observation" | "answer" | "error";
  content: string;
  step: number;
  tool_args?: string;
  timestamp: string;
}

export function useSSE() {
  const [events, setEvents] = useState<StepEvent[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  const sendMessage = useCallback(
    async (sessionId: string, query: string) => {
      setIsStreaming(true);
      setEvents([]);

      const controller = new AbortController();
      abortRef.current = controller;

      try {
        const resp = await fetch("/api/v1/chat/stream", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ session_id: sessionId, query }),
          signal: controller.signal,
        });

        const reader = resp.body?.getReader();
        if (!reader) return;

        const decoder = new TextDecoder();
        let buffer = "";

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() || "";

          for (const line of lines) {
            if (line.startsWith("data: ")) {
              try {
                const data: StepEvent = JSON.parse(line.slice(6));
                setEvents((prev) => [...prev, data]);
              } catch {
                // skip parse errors
              }
            }
          }
        }
      } catch (err: unknown) {
        if (err instanceof Error && err.name === "AbortError") return;
        setEvents((prev) => [
          ...prev,
          {
            type: "error",
            content: `连接失败: ${err instanceof Error ? err.message : String(err)}`,
            step: 0,
            timestamp: new Date().toISOString(),
          },
        ]);
      } finally {
        setIsStreaming(false);
      }
    },
    []
  );

  const cancel = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  return { events, isStreaming, sendMessage, cancel };
}
