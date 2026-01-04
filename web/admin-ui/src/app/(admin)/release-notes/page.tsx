"use client";

import { useEffect, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { fetchReleaseNotes } from "@/lib/admin-api";
import type { ReleaseNotesResponse } from "@/lib/admin-types";

export default function ReleaseNotesPage() {
  const [data, setData] = useState<ReleaseNotesResponse | null>(null);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    let active = true;
    fetchReleaseNotes()
      .then((payload) => {
        if (!active) return;
        setData(payload);
      })
      .catch((err) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : "failed to load release notes");
      });
    return () => {
      active = false;
    };
  }, []);

  const entries = data ? Object.values(data.clis) : [];

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <Badge>Release Notes</Badge>
          <CardTitle className="mt-3 text-2xl">CLI 版本动态</CardTitle>
          <CardDescription>
            汇总各 CLI 的最新版本与本地版本，及时发现更新。
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted">
            {data ? `最后刷新：${data.last_updated}` : "正在拉取最新信息..."}
          </p>
          {error ? <p className="mt-2 text-sm text-red-500">{error}</p> : null}
        </CardContent>
      </Card>

      <div className="grid gap-6 lg:grid-cols-2">
        {entries.map((entry) => (
          <Card key={entry.cli_name}>
            <CardHeader>
              <CardTitle className="text-xl">{entry.display_name}</CardTitle>
              <CardDescription>CLI 名称：{entry.cli_name}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 text-sm text-muted">
                <div className="flex items-center justify-between">
                  <span>最新版本</span>
                  <span className="font-semibold text-ink">{entry.latest_version}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span>本地版本</span>
                  <span>{entry.local_version || "--"}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span>更新状态</span>
                  <span className={entry.update_available ? "text-accent" : "text-ink"}>
                    {entry.update_available ? "可更新" : "已是最新"}
                  </span>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
