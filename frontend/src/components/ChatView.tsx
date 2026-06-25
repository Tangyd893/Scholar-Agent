import { useRef, useEffect } from "react";
import type { StepEvent } from "../hooks/useSSE";
import { ThoughtChain } from "./ThoughtChain";
import { cn } from "../lib/utils";

interface ChatViewProps {
  messages: Message[];
  currentEvents: StepEvent[];
  isStreaming: boolean;
}

export interface Message {
  role: "user" | "assistant";
  content: string;
  events?: StepEvent[];
}

export function ChatView({ messages, currentEvents, isStreaming }: ChatViewProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, currentEvents]);

  const allMessages = [...messages];
  if (isStreaming && currentEvents.length > 0) {
    const answerEvent = currentEvents.find((e) => e.type === "answer");
    allMessages.push({
      role: "assistant",
      content: answerEvent?.content || "思考中...",
      events: currentEvents,
    });
  }

  if (allMessages.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center text-zinc-600">
        <div className="text-center">
          <p className="text-lg mb-2">🎓 ScholarAgent</p>
          <p className="text-sm">输入问题，开始学术探索</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto px-4 py-4 space-y-4">
      {allMessages.map((msg, i) => (
        <div
          key={i}
          className={cn(
            "max-w-3xl mx-auto",
            msg.role === "user" ? "flex justify-end" : "flex justify-start"
          )}
        >
          <div
            className={cn(
              "rounded-lg px-4 py-3 max-w-[85%]",
              msg.role === "user"
                ? "bg-scholar-600 text-white"
                : "bg-zinc-900 text-zinc-200 border border-zinc-800"
            )}
          >
            {msg.events && msg.events.length > 0 && (
              <ThoughtChain events={msg.events} />
            )}
            <p className="whitespace-pre-wrap text-sm leading-relaxed">
              {msg.content}
            </p>
          </div>
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  );
}
