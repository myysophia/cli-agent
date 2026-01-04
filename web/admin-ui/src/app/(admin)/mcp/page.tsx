"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  createClaudeMCP,
  createCursorMCP,
  deleteClaudeMCP,
  deleteCursorMCP,
  fetchClaudeMCP,
  fetchCursorMCP,
  updateClaudeMCP,
  updateCursorMCP
} from "@/lib/admin-api";
import type { AdminMCPMeta, AdminMCPServer } from "@/lib/admin-types";

const emptyServer: AdminMCPServer = {
  name: "",
  command: "",
  args: [],
  env: []
};

export default function MCPPage() {
  const [provider, setProvider] = useState<"cursor" | "claude">("cursor");
  const [servers, setServers] = useState<AdminMCPServer[]>([]);
  const [meta, setMeta] = useState<AdminMCPMeta | null>(null);
  const [selectedName, setSelectedName] = useState<string>("");
  const [form, setForm] = useState<AdminMCPServer>(emptyServer);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<"create" | "update" | "delete" | null>(null);

  const selectedServer = useMemo(
    () => servers.find((item) => item.name === selectedName) || null,
    [servers, selectedName]
  );

  const loadServers = useCallback(async () => {
    setLoading(true);
    setError("");
    setMeta(null);
    setServers([]);
    try {
      const payload = provider === "cursor" ? await fetchCursorMCP() : await fetchClaudeMCP();
      setServers(payload.servers || []);
      setMeta(payload.meta);
      if (!selectedName && payload.servers.length > 0) {
        setSelectedName(payload.servers[0].name);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "读取 MCP 配置失败");
    } finally {
      setLoading(false);
    }
  }, [provider, selectedName]);

  useEffect(() => {
    loadServers();
  }, [loadServers]);

  useEffect(() => {
    if (!selectedName) {
      setForm(emptyServer);
      return;
    }
    const server = servers.find((item) => item.name === selectedName);
    if (server) {
      setForm({
        ...server,
        args: server.args || [],
        env: server.env || []
      });
    }
  }, [selectedName, servers]);

  const handleSelect = (name: string) => {
    setMessage("");
    setSelectedName(name);
  };

  const handleNew = () => {
    setMessage("");
    setSelectedName("");
    setForm(emptyServer);
  };

  const updateField = (key: keyof AdminMCPServer, value: AdminMCPServer[keyof AdminMCPServer]) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const addArg = () => {
    updateField("args", [...(form.args || []), ""]);
  };

  const updateArg = (index: number, value: string) => {
    const next = [...(form.args || [])];
    next[index] = value;
    updateField("args", next);
  };

  const removeArg = (index: number) => {
    updateField(
      "args",
      (form.args || []).filter((_, idx) => idx !== index)
    );
  };

  const addEnv = () => {
    updateField("env", [...(form.env || []), { key: "", value: "", masked: false }]);
  };

  const updateEnv = (
    index: number,
    updater: (item: { key: string; value: string; masked: boolean }) => { key: string; value: string; masked: boolean }
  ) => {
    setForm((prev) => {
      const env = [...(prev.env || [])];
      env[index] = updater(env[index]);
      return { ...prev, env };
    });
  };

  const removeEnv = (index: number) => {
    updateField(
      "env",
      (form.env || []).filter((_, idx) => idx !== index)
    );
  };

  const requestSave = () => {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      if (!form.name.trim()) {
        throw new Error("请填写 MCP 名称");
      }
      if (!form.command.trim()) {
        throw new Error("请填写 command");
      }
      setPendingAction(selectedName ? "update" : "create");
      setConfirmOpen(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  const performSave = async () => {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      const payload: AdminMCPServer = {
        ...form,
        name: form.name.trim(),
        command: form.command.trim(),
        args: (form.args || []).map((item) => item.trim()).filter(Boolean),
        env: (form.env || []).filter((item) => item.key.trim() !== "")
      };
      const saved =
        provider === "cursor"
          ? selectedName
            ? await updateCursorMCP(selectedName, payload)
            : await createCursorMCP(payload)
          : selectedName
            ? await updateClaudeMCP(selectedName, payload)
            : await createClaudeMCP(payload);
      setSelectedName(saved.name);
      setForm({
        ...saved,
        args: saved.args || [],
        env: saved.env || []
      });
      await loadServers();
      setMessage("保存成功");
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  const requestDelete = () => {
    if (!selectedName) return;
    setPendingAction("delete");
    setConfirmOpen(true);
  };

  const performDelete = async () => {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      if (provider === "cursor") {
        await deleteCursorMCP(selectedName);
      } else {
        await deleteClaudeMCP(selectedName);
      }
      setSelectedName("");
      setForm(emptyServer);
      await loadServers();
      setMessage("已删除");
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除失败");
    } finally {
      setSaving(false);
    }
  };

  const handleConfirm = async () => {
    if (!pendingAction) return;
    setConfirmOpen(false);
    if (pendingAction === "delete") {
      await performDelete();
    } else {
      await performSave();
    }
    setPendingAction(null);
  };

  const confirmTitle = pendingAction === "delete" ? "确认删除" : pendingAction === "update" ? "确认更新" : "确认创建";
  const confirmName = selectedName || form.name.trim();
  const confirmDescription =
    pendingAction === "delete"
      ? `将删除 MCP Server "${confirmName}"，此操作不可恢复。`
      : pendingAction === "update"
        ? `将更新 MCP Server "${confirmName}" 并写入配置。`
        : `将创建 MCP Server "${confirmName}" 并写入配置。`;

  return (
    <div className="grid gap-6 lg:grid-cols-[320px_1fr]">
      <Card>
        <CardHeader>
          <Badge>MCP</Badge>
          <CardTitle className="mt-3 text-xl">
            {provider === "cursor" ? "Cursor MCP" : "Claude MCP"}
          </CardTitle>
          <CardDescription>
            当前读取：{meta?.display_path || "--"}（{meta?.exists ? "已存在" : "未找到"}）
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid grid-cols-2 gap-2">
            <Button
              variant={provider === "cursor" ? "solid" : "outline"}
              size="sm"
              onClick={() => {
                setProvider("cursor");
                setSelectedName("");
                setForm(emptyServer);
              }}
            >
              Cursor
            </Button>
            <Button
              variant={provider === "claude" ? "solid" : "outline"}
              size="sm"
              onClick={() => {
                setProvider("claude");
                setSelectedName("");
                setForm(emptyServer);
              }}
            >
              Claude
            </Button>
          </div>
          <Button size="sm" className="w-full" onClick={handleNew}>
            新增 MCP Server
          </Button>
          <div className="space-y-2">
            {servers.length === 0 ? (
              <p className="text-xs text-muted">暂无 MCP 配置</p>
            ) : (
              servers.map((server) => (
                <button
                  key={server.name}
                  type="button"
                  onClick={() => handleSelect(server.name)}
                  className={`w-full rounded-2xl border px-4 py-3 text-left text-sm transition ${
                    server.name === selectedName ? "border-accent/40 bg-accent/10" : "border-black/5 bg-white/70"
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-ink">{server.name}</span>
                  </div>
                  <p className="mt-1 text-xs text-muted">{server.command || "--"}</p>
                </button>
              ))
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <Badge>编辑</Badge>
          <CardTitle className="mt-3 text-xl">
            {selectedName ? `编辑 ${selectedName}` : "新建 MCP Server"}
          </CardTitle>
          <CardDescription>敏感 env 会掩码显示，保持不变请勿修改。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {error ? <p className="text-sm text-red-500">{error}</p> : null}
          {message ? <p className="text-sm text-emerald-600">{message}</p> : null}

          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <p className="text-xs text-muted">名称</p>
              <Input
                value={form.name}
                onChange={(event) => updateField("name", event.target.value)}
                disabled={!!selectedName}
                placeholder="例如: fetch"
              />
            </div>
            <div className="space-y-2">
              <p className="text-xs text-muted">Command</p>
              <Input
                value={form.command}
                onChange={(event) => updateField("command", event.target.value)}
                placeholder="例如: mcp-server-fetch"
              />
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-xs text-muted">Args</p>
              <Button variant="outline" size="sm" onClick={addArg}>
                添加参数
              </Button>
            </div>
            {(form.args || []).length === 0 ? (
              <p className="text-xs text-muted">暂无参数</p>
            ) : (
              <div className="space-y-2">
                {(form.args || []).map((arg, index) => (
                  <div key={`arg-${index}`} className="flex gap-2">
                    <Input value={arg} onChange={(event) => updateArg(index, event.target.value)} />
                    <Button variant="outline" size="sm" onClick={() => removeArg(index)}>
                      删除
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-xs text-muted">Env</p>
              <Button variant="outline" size="sm" onClick={addEnv}>
                添加变量
              </Button>
            </div>
            {(form.env || []).length === 0 ? (
              <p className="text-xs text-muted">暂无环境变量</p>
            ) : (
              <div className="space-y-2">
                {(form.env || []).map((item, index) => (
                  <div key={`env-${index}`} className="grid gap-2 md:grid-cols-[1fr_2fr_auto]">
                    <Input
                      value={item.key}
                      onChange={(event) =>
                        updateEnv(index, (prev) => ({ ...prev, key: event.target.value }))
                      }
                      placeholder="KEY"
                    />
                    <Input
                      value={item.value}
                      onChange={(event) =>
                        updateEnv(index, (prev) => ({
                          ...prev,
                          value: event.target.value,
                          masked: prev.masked ? false : prev.masked
                        }))
                      }
                      placeholder="VALUE"
                    />
                    <Button variant="outline" size="sm" onClick={() => removeEnv(index)}>
                      删除
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="flex flex-wrap gap-3">
            <Button onClick={requestSave} disabled={saving || loading}>
              {saving ? "保存中..." : selectedName ? "保存修改" : "创建 MCP"}
            </Button>
            {selectedName ? (
              <Button variant="outline" onClick={requestDelete} disabled={saving}>
                删除 MCP
              </Button>
            ) : null}
          </div>

          {selectedServer ? (
            <p className="text-xs text-muted">
              当前 env 数量：{selectedServer.env?.length || 0}，args：{selectedServer.args?.length || 0}
            </p>
          ) : null}
        </CardContent>
      </Card>

      <Dialog
        open={confirmOpen}
        onOpenChange={(open) => {
          setConfirmOpen(open);
          if (!open) {
            setPendingAction(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{confirmTitle}</DialogTitle>
            <DialogDescription>{confirmDescription}</DialogDescription>
          </DialogHeader>
          <DialogFooter className="pt-4">
            <Button variant="outline" onClick={() => setConfirmOpen(false)}>
              取消
            </Button>
            <Button onClick={handleConfirm} disabled={saving}>
              {pendingAction === "delete" ? "确认删除" : "确认继续"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
