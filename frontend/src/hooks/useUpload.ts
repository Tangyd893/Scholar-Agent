import { useState, useCallback, useRef } from "react";

export interface UploadJob {
  job_id: string;
  status: "pending" | "processing" | "completed" | "failed";
  progress: number;
  error?: string;
}

export function useUpload() {
  const [job, setJob] = useState<UploadJob | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const pollRef = useRef<number | null>(null);

  const uploadPDF = useCallback(async (file: File, sessionId: string) => {
    setIsUploading(true);
    setJob(null);

    const formData = new FormData();
    formData.append("file", file);
    if (sessionId) formData.append("session_id", sessionId);

    try {
      const resp = await fetch("/api/v1/papers/upload", {
        method: "POST",
        body: formData,
      });
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error || "upload failed");

      setJob({ job_id: data.job_id, status: "pending", progress: 0 });

      // 轮询进度
      pollRef.current = window.setInterval(async () => {
        try {
          const jResp = await fetch(`/api/v1/jobs/${data.job_id}`);
          const jData: UploadJob = await jResp.json();
          setJob(jData);
          if (jData.status === "completed" || jData.status === "failed") {
            if (pollRef.current) clearInterval(pollRef.current);
            setIsUploading(false);
          }
        } catch {
          // retry next poll
        }
      }, 1000);
    } catch (err) {
      setJob({
        job_id: "",
        status: "failed",
        progress: 0,
        error: err instanceof Error ? err.message : "unknown error",
      });
      setIsUploading(false);
    }
  }, []);

  const clearJob = useCallback(() => {
    if (pollRef.current) clearInterval(pollRef.current);
    setJob(null);
    setIsUploading(false);
  }, []);

  return { job, isUploading, uploadPDF, clearJob };
}
