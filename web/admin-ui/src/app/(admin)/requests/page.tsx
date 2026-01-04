"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { callChat } from "@/lib/admin-api";
import type { ChatRequest } from "@/lib/admin-types";

export default function RequestsPage() {
  const [prompt, setPrompt] = useState("");
  const [profile, setProfile] = useState("");
  const [cli, setCli] = useState("");
  const [workflowRunId, setWorkflowRunId] = useState("");
  const [result, setResult] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!prompt.trim()) {
      return;
    }
    setLoading(true);
    setError("");
    setResult("");

    try {
      const payload: ChatRequest = { prompt };
      if (profile.trim()) {
        payload.profile = profile.trim();
      }
      if (cli.trim()) {
        payload.cli = cli.trim();
      }
      if (workflowRunId.trim()) {
        payload.workflow_run_id = workflowRunId.trim();
      }
      const response = await callChat(payload);
      setResult(JSON.stringify(response, null, 2));
    } catch (err) {
      setError(err instanceof Error ? err.message : "request failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="grid gap-6 lg:grid-cols-[1.2fr_1fr]">
      <Card>
        <CardHeader>
          <CardTitle>请求实验室</CardTitle>
          <CardDescription>快速调用 /chat 接口进行验证。</CardDescription>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={handleSubmit}>
            <Input
              placeholder="Prompt"
              value={prompt}
              onChange={(event) => setPrompt(event.target.value)}
            />
            <Input
              placeholder="Profile（可选）"
              value={profile}
              onChange={(event) => setProfile(event.target.value)}
            />
            <Input
              placeholder="CLI（可选）"
              value={cli}
              onChange={(event) => setCli(event.target.value)}
            />
            <Input
              placeholder="workflow_run_id（可选）"
              value={workflowRunId}
              onChange={(event) => setWorkflowRunId(event.target.value)}
            />
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "请求中..." : "发送请求"}
            </Button>
          </form>
          {error ? <p className="mt-4 text-sm text-red-500">{error}</p> : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>响应结果</CardTitle>
          <CardDescription>返回的 JSON 将显示在下方。</CardDescription>
        </CardHeader>
        <CardContent>
          <pre className="max-h-[420px] overflow-auto rounded-2xl border border-black/5 bg-bg-soft/70 p-4 text-xs text-ink">
            {result || "尚未发送请求。"}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}
