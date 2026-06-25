import { useState, useCallback, useEffect, useRef } from "react";
import { Plus } from "lucide-react";
import { ChatView, type Message } from "./components/ChatView";
import { MessageInput } from "./components/MessageInput";
import { useSSE } from "./hooks/useSSE";
import { useUpload } from "./hooks/useUpload";

function generateSessionId() {
  return `sess_${Date.now()}${Math.random().toString(36).slice(2, 9)}`;
}

export default function App() {
  const [sessionId, setSessionId] = useState(() => generateSessionId());
  const [messages, setMessages] = useState<Message[]>([]);
  const { events, isStreaming, sendMessage, cancel } = useSSE();
  const { job: uploadJob, isUploading, uploadPDF, clearJob } = useUpload();
  const prevStreaming = useRef(isStreaming);

  const handleSend = useCallback(
    async (query: string) => {
      const userMsg: Message = { role: "user", content: query };
      setMessages((prev) => [...prev, userMsg]);
      await sendMessage(sessionId, query);
    },
    [sessionId, sendMessage]
  );

  // Capture answer event when stream ends
  useEffect(() => {
    if (!isStreaming && prevStreaming.current && events.length > 0) {
      const answerEvent = events.find((e) => e.type === "answer");
      if (answerEvent) {
        const steps = events.filter((e) => e.type !== "answer");
        setMessages((prev) => [
          ...prev,
          {
            role: "assistant" as const,
            content: answerEvent.content,
            events: steps,
          },
        ]);
      }
    }
    prevStreaming.current = isStreaming;
  }, [isStreaming, events]);

  const handleNewSession = () => {
    setSessionId(generateSessionId());
    setMessages([]);
  };

  return (
    <div className="h-dvh flex flex-col bg-zinc-950">
      <header className="border-b border-zinc-800 px-4 py-3 flex items-center justify-between shrink-0">
        <h1 className="text-lg font-semibold text-zinc-200 tracking-tight">
          🎓 ScholarAgent
        </h1>
        <button
          onClick={handleNewSession}
          className="flex items-center gap-1.5 text-sm text-zinc-400 hover:text-zinc-200 transition-colors px-3 py-1.5 rounded-md hover:bg-zinc-800"
        >
          <Plus className="w-4 h-4" />
          新会话
        </button>
      </header>

      <ChatView
        messages={messages}
        currentEvents={isStreaming ? events : []}
        isStreaming={isStreaming}
      />

      <MessageInput
        onSend={handleSend}
        isStreaming={isStreaming}
        onCancel={cancel}
        uploadJob={uploadJob}
        isUploading={isUploading}
        onUpload={(file) => uploadPDF(file, sessionId)}
        onClearUpload={clearJob}
        sessionId={sessionId}
      />
    </div>
  );
}
