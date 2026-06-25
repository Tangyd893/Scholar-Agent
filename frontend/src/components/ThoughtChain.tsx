import { useState } from "react";
import { ChevronDown, ChevronRight, Brain, Wrench, Eye, AlertTriangle } from "lucide-react";
import type { StepEvent } from "../hooks/useSSE";
import { cn } from "../lib/utils";

interface ThoughtChainProps {
  events: StepEvent[];
}

const iconMap: Record<string, React.ReactNode> = {
  thought: <Brain className="w-4 h-4 text-amber-400" />,
  action: <Wrench className="w-4 h-4 text-blue-400" />,
  observation: <Eye className="w-4 h-4 text-emerald-400" />,
  error: <AlertTriangle className="w-4 h-4 text-red-400" />,
};

const labelMap: Record<string, string> = {
  thought: "思考",
  action: "行动",
  observation: "观察",
  error: "错误",
};

export function ThoughtChain({ events }: ThoughtChainProps) {
  const [expanded, setExpanded] = useState(true);

  if (events.length === 0) return null;

  return (
    <div className="my-2 border border-zinc-800 rounded-lg overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center gap-2 px-3 py-2 bg-zinc-900 hover:bg-zinc-800 transition-colors text-sm text-zinc-400"
      >
        {expanded ? (
          <ChevronDown className="w-3.5 h-3.5" />
        ) : (
          <ChevronRight className="w-3.5 h-3.5" />
        )}
        <span>推理链 ({events.length} 步)</span>
      </button>

      {expanded && (
        <div className="divide-y divide-zinc-800">
          {events.map((event, i) => (
            <div key={i} className="px-3 py-2 text-sm">
              <div className="flex items-center gap-2 mb-1">
                {iconMap[event.type]}
                <span className="font-medium text-zinc-300">
                  {labelMap[event.type] || event.type}
                </span>
                <span className="text-zinc-600 text-xs">
                  Step {event.step}
                </span>
              </div>
              <div
                className={cn(
                  "text-zinc-400 whitespace-pre-wrap break-words",
                  event.type === "action" && "text-blue-300"
                )}
              >
                {event.type === "action" ? (
                  <span>
                    <span className="font-mono text-blue-300">
                      {event.content}
                    </span>
                    {event.tool_args && (
                      <span className="text-zinc-500 ml-1 font-mono text-xs">
                        {event.tool_args}
                      </span>
                    )}
                  </span>
                ) : (
                  <span>
                    {event.content.length > 300
                      ? event.content.slice(0, 300) + "..."
                      : event.content}
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
