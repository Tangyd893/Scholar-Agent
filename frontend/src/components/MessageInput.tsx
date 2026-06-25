import { useState, useRef, type FormEvent, type ChangeEvent } from "react";
import { Send, FileText, Loader2, CheckCircle, XCircle, Upload } from "lucide-react";
import type { UploadJob } from "../hooks/useUpload";
import { cn } from "../lib/utils";

interface MessageInputProps {
  onSend: (query: string) => void;
  isStreaming: boolean;
  onCancel: () => void;
  uploadJob: UploadJob | null;
  isUploading: boolean;
  onUpload: (file: File) => void;
  onClearUpload: () => void;
  sessionId: string;
}

export function MessageInput({
  onSend,
  isStreaming,
  onCancel,
  uploadJob,
  isUploading,
  onUpload,
  onClearUpload,
}: MessageInputProps) {
  const [input, setInput] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileRef = useRef<HTMLInputElement>(null);

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

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) onUpload(file);
    if (fileRef.current) fileRef.current.value = "";
  };

  const progressPercent = uploadJob?.progress ?? 0;

  return (
    <form onSubmit={handleSubmit} className="border-t border-zinc-800 bg-zinc-950">
      {/* Upload progress bar */}
      {uploadJob && (
        <div className="px-4 py-2 max-w-3xl mx-auto">
          <div className="flex items-center gap-2 text-sm">
            {uploadJob.status === "completed" ? (
              <CheckCircle className="w-4 h-4 text-emerald-400" />
            ) : uploadJob.status === "failed" ? (
              <XCircle className="w-4 h-4 text-red-400" />
            ) : (
              <Upload className="w-4 h-4 text-blue-400 animate-pulse" />
            )}
            <span className="text-zinc-300">
              {uploadJob.status === "completed"
                ? "PDF 解析完成，可开始 RAG 问答"
                : uploadJob.status === "failed"
                ? `解析失败: ${uploadJob.error || "未知错误"}`
                : `解析中... ${progressPercent}%`}
            </span>
            {(uploadJob.status === "completed" ||
              uploadJob.status === "failed") && (
              <button
                type="button"
                onClick={onClearUpload}
                className="ml-auto text-xs text-zinc-500 hover:text-zinc-300"
              >
                关闭
              </button>
            )}
          </div>
          {uploadJob.status !== "completed" && uploadJob.status !== "failed" && (
            <div className="mt-1 h-1 bg-zinc-800 rounded-full overflow-hidden">
              <div
                className="h-full bg-scholar-500 transition-all duration-500"
                style={{ width: `${Math.max(progressPercent, 5)}%` }}
              />
            </div>
          )}
        </div>
      )}

      {/* Input row */}
      <div className="px-4 py-3">
        <div className="max-w-3xl mx-auto flex items-end gap-2">
          <input
            ref={fileRef}
            type="file"
            accept=".pdf"
            onChange={handleFileChange}
            className="hidden"
          />
          <button
            type="button"
            onClick={() => fileRef.current?.click()}
            disabled={isUploading}
            className={cn(
              "p-2 rounded-lg transition-colors",
              isUploading
                ? "text-blue-400 bg-blue-400/10"
                : "text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800"
            )}
            title="上传 PDF"
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
              placeholder={
                uploadJob?.status === "completed"
                  ? "PDF 已就绪，输入问题开始 RAG 问答..."
                  : "输入学术问题，如：帮我找 attention 相关的经典论文"
              }
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
      </div>
    </form>
  );
}
