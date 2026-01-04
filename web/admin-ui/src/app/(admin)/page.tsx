"use client";

import { useEffect, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { MetricCard } from "@/components/metric-card";
import { fetchAdminHealth } from "@/lib/admin-api";
import type { AdminHealthResponse } from "@/lib/admin-types";

export default function DashboardPage() {
  const [health, setHealth] = useState<AdminHealthResponse | null>(null);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    let active = true;
    fetchAdminHealth()
      .then((data) => {
        if (!active) return;
        setHealth(data);
      })
      .catch((err) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : "health check failed");
      });
    return () => {
      active = false;
    };
  }, []);

  return (
    <div className="space-y-8">
      <Card className="fade-up">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <Badge>系统概览</Badge>
              <CardTitle className="mt-3 text-3xl">CLI Gateway 控制台</CardTitle>
              <CardDescription>监控服务状态、调用趋势与工作流健康度。</CardDescription>
            </div>
            <div className="rounded-2xl border border-black/5 bg-bg-soft/80 px-4 py-3 text-sm text-muted">
              {health ? `最新心跳 ${health.time}` : "等待健康检查..."}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap items-center gap-3 text-sm">
            <span className="text-muted">Health</span>
            <span className="rounded-full bg-ink px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-white">
              {health?.status ?? (error ? "error" : "pending")}
            </span>
            {error ? <span className="text-xs text-red-500">{error}</span> : null}
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-6 lg:grid-cols-3">
        <MetricCard title="请求总量" value="--" description="等待接入调用统计。" />
        <MetricCard title="成功率" value="--" description="请求成功/失败比率。" />
        <MetricCard title="平均延迟" value="--" description="P50 / P95 延迟趋势。" />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>工作流会话</CardTitle>
            <CardDescription>Redis 映射、锁状态与最新活动。</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3 text-sm text-muted">
              <p>连接状态：等待 Admin API 扩展</p>
              <p>最近映射：--</p>
              <p>锁等待：--</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>环境概览</CardTitle>
            <CardDescription>运行配置与默认 profile 摘要。</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3 text-sm text-muted">
              <p>默认 Profile：--</p>
              <p>启用 CLI：claude / codex / cursor / gemini / qwen</p>
              <p>后台路由：/v1/admin</p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
