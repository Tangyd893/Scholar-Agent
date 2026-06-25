import { useState, useRef, type FormEvent } from "react";
import { Send, FileText, Loader2 } from "lucide-react";
import { cn } from "../lib/utils";

interface MessageInputProps {
  onSend: (query: string) => void;
  isStreaming: boolean;
  onCancel: () => void;
}

export function MessageInput({ onSend, isStreaming, onCancel }: MessageInputProps) {
  const [input, setInput] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isStreaming) return;
    onSend(input.trim());
    setInput("");
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  const handleInput = () => {
    const el = textareaRef.current;
    if (el) {
      el.style.height = "auto";
      el.style.height = Math.min(el.scrollHeight, 200) + "px";
    }
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="border-t border-zinc-800 bg-zinc-950 px-4 py-3"
    >
      <div className="max-w-3xl mx-auto flex items-end gap-2">
        <button
          type="button"
          className="p-2 text-zinc-500 hover:text-zinc-300 transition-colors"
          title="上传 PDF（即将推出）"
        >
          <FileText className="w-5 h-5" />
        </button>

        <div className="flex-1 relative">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onInput={handleInput}
            onKeyDown={handleKeyDown}
            placeholder="输入学术问题，如：帮我找 attention 相关的经典论文"
            rows={1}
            disabled={isStreaming}
            className={cn(
              "w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 pr-10",
              "text-sm text-zinc-200 placeholder:text-zinc-600",
              "resize-none focus:outline-none focus:border-scholar-500",
              "disabled:opacity-50"
            )}
          />
        </div>

        {isStreaming ? (
          <button
            type="button"
            onClick={onCancel}
            className="p-2 text-red-400 hover:text-red-300 transition-colors"
            title="停止生成"
          >
            <Loader2 className="w-5 h-5 animate-spin" />
          </button>
        ) : (
          <button
            type="submit"
            disabled={!input.trim()}
            className={cn(
              "p-2 rounded-lg transition-colors",
              input.trim()
                ? "bg-scholar-600 text-white hover:bg-scholar-500"
                : "bg-zinc-800 text-zinc-600"
            )}
          >
            <Send className="w-5 h-5" />
          </button>
        )}
      </div>
    </form>
  );
}
