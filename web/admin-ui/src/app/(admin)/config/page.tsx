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
  createProfile,
  deleteProfile,
  fetchProfile,
  fetchProfiles,
  updateProfile
} from "@/lib/admin-api";
import type {
  AdminProfileEnvItem,
  AdminProfilePayload,
  AdminProfileSummary
} from "@/lib/admin-types";

const emptyProfile: AdminProfilePayload = {
  key: "",
  name: "",
  cli: "",
  model: "",
  allowed_tools: [],
  skills: [],
  system_prompt: "",
  system_prompt_masked: false,
  env: [],
  is_default: false
};

export default function ConfigPage() {
  const [profiles, setProfiles] = useState<AdminProfileSummary[]>([]);
  const [selectedKey, setSelectedKey] = useState<string>("");
  const [form, setForm] = useState<AdminProfilePayload>(emptyProfile);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<"create" | "update" | "delete" | null>(null);

  const selectedSummary = useMemo(
    () => profiles.find((item) => item.key === selectedKey) || null,
    [profiles, selectedKey]
  );

  const loadProfiles = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const payload = await fetchProfiles();
      setProfiles(payload.profiles || []);
      const defaultKey =
        payload.profiles.find((profile) => profile.is_default)?.key || payload.profiles[0]?.key || "";
      if (!selectedKey && defaultKey) {
        setSelectedKey(defaultKey);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "读取 profiles 失败");
    } finally {
      setLoading(false);
    }
  }, [selectedKey]);

  const loadProfile = async (key: string) => {
    if (!key) {
      setForm(emptyProfile);
      return;
    }
    setLoading(true);
    setError("");
    try {
      const payload = await fetchProfile(key);
      setForm({
        ...payload,
        env: payload.env || [],
        skills: payload.skills || [],
        allowed_tools: payload.allowed_tools || []
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "读取 profile 失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProfiles();
  }, [loadProfiles]);

  useEffect(() => {
    if (selectedKey) {
      loadProfile(selectedKey);
    }
  }, [selectedKey]);

  const handleSelect = (key: string) => {
    setMessage("");
    setSelectedKey(key);
  };

  const handleNew = () => {
    setMessage("");
    setSelectedKey("");
    setForm(emptyProfile);
  };

  const updateField = (key: keyof AdminProfilePayload, value: AdminProfilePayload[keyof AdminProfilePayload]) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const updateEnvItem = (index: number, updater: (item: AdminProfileEnvItem) => AdminProfileEnvItem) => {
    setForm((prev) => {
      const env = [...(prev.env || [])];
      env[index] = updater(env[index]);
      return { ...prev, env };
    });
  };

  const addEnvItem = () => {
    setForm((prev) => ({
      ...prev,
      env: [...(prev.env || []), { key: "", value: "", masked: false }]
    }));
  };

  const removeEnvItem = (index: number) => {
    setForm((prev) => ({
      ...prev,
      env: (prev.env || []).filter((_, idx) => idx !== index)
    }));
  };

  const addSkill = () => {
    updateField("skills", [...(form.skills || []), ""]);
  };

  const updateSkill = (index: number, value: string) => {
    const next = [...(form.skills || [])];
    next[index] = value;
    updateField("skills", next);
  };

  const removeSkill = (index: number) => {
    updateField(
      "skills",
      (form.skills || []).filter((_, idx) => idx !== index)
    );
  };

  const addTool = () => {
    updateField("allowed_tools", [...(form.allowed_tools || []), ""]);
  };

  const updateTool = (index: number, value: string) => {
    const next = [...(form.allowed_tools || [])];
    next[index] = value;
    updateField("allowed_tools", next);
  };

  const removeTool = (index: number) => {
    updateField(
      "allowed_tools",
      (form.allowed_tools || []).filter((_, idx) => idx !== index)
    );
  };

  const requestSave = async () => {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      if (!form.key.trim()) {
        throw new Error("请填写 profile key");
      }
      if (!form.name.trim()) {
        throw new Error("请填写显示名称");
      }
      setPendingAction(selectedKey ? "update" : "create");
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
      const payload: AdminProfilePayload = {
        ...form,
        key: form.key.trim(),
        name: form.name.trim(),
        cli: form.cli?.trim() || "",
        model: form.model?.trim() || "",
        allowed_tools: (form.allowed_tools || []).map((item) => item.trim()).filter(Boolean),
        skills: (form.skills || []).map((item) => item.trim()).filter(Boolean),
        env: (form.env || []).filter((item) => item.key.trim() !== "")
      };

      let saved: AdminProfilePayload;
      if (!selectedKey) {
        saved = await createProfile(payload);
        setSelectedKey(saved.key);
      } else {
        saved = await updateProfile(selectedKey, payload);
      }
      setForm({
        ...saved,
        env: saved.env || [],
        skills: saved.skills || [],
        allowed_tools: saved.allowed_tools || []
      });
      await loadProfiles();
      setMessage("保存成功");
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存失败");
    } finally {
      setSaving(false);
    }
  };

  const requestDelete = () => {
    if (!selectedKey) return;
    setPendingAction("delete");
    setConfirmOpen(true);
  };

  const performDelete = async () => {
    setSaving(true);
    setError("");
    setMessage("");
    try {
      await deleteProfile(selectedKey);
      setSelectedKey("");
      setForm(emptyProfile);
      await loadProfiles();
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
  const confirmKey = selectedKey || form.key.trim();
  const confirmDescription =
    pendingAction === "delete"
      ? `将删除 profile "${confirmKey}"，此操作不可恢复。`
      : pendingAction === "update"
        ? `将更新 profile "${confirmKey}" 并热加载配置。`
        : `将创建 profile "${confirmKey}" 并写入配置。`;

  return (
    <div className="grid gap-6 lg:grid-cols-[320px_1fr]">
      <Card>
        <CardHeader>
          <Badge>Profiles</Badge>
          <CardTitle className="mt-3 text-xl">配置列表</CardTitle>
          <CardDescription>点击条目进入编辑，支持新增/删除。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <Button size="sm" className="w-full" onClick={handleNew}>
            新增 Profile
          </Button>
          <div className="space-y-2">
            {profiles.length === 0 ? (
              <p className="text-xs text-muted">暂无 profiles</p>
            ) : (
              profiles.map((profile) => (
                <button
                  key={profile.key}
                  type="button"
                  onClick={() => handleSelect(profile.key)}
                  className={`w-full rounded-2xl border px-4 py-3 text-left text-sm transition ${
                    profile.key === selectedKey ? "border-accent/40 bg-accent/10" : "border-black/5 bg-white/70"
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-ink">{profile.key}</span>
                    {profile.is_default ? <Badge className="border-accent/40">默认</Badge> : null}
                  </div>
                  <p className="mt-1 text-xs text-muted">{profile.name || "--"}</p>
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
            {selectedKey ? `编辑 ${selectedKey}` : "新建 Profile"}
          </CardTitle>
          <CardDescription>Env 等敏感字段会掩码显示，保持不变请勿修改。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {error ? <p className="text-sm text-red-500">{error}</p> : null}
          {message ? <p className="text-sm text-emerald-600">{message}</p> : null}

          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <p className="text-xs text-muted">Profile Key</p>
              <Input
                value={form.key}
                onChange={(event) => updateField("key", event.target.value)}
                disabled={!!selectedKey}
                placeholder="例如: codex"
              />
            </div>
            <div className="space-y-2">
              <p className="text-xs text-muted">显示名称</p>
              <Input
                value={form.name}
                onChange={(event) => updateField("name", event.target.value)}
                placeholder="例如: OpenAI Codex"
              />
            </div>
            <div className="space-y-2">
              <p className="text-xs text-muted">CLI</p>
              <Input
                value={form.cli || ""}
                onChange={(event) => updateField("cli", event.target.value)}
                placeholder="claude / codex / cursor ..."
              />
            </div>
            <div className="space-y-2">
              <p className="text-xs text-muted">模型</p>
              <Input
                value={form.model || ""}
                onChange={(event) => updateField("model", event.target.value)}
                placeholder="gpt-5.1 / sonnet-4 ..."
              />
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <p className="text-xs text-muted">System Prompt</p>
              <label className="flex items-center gap-2 text-xs text-muted">
                <input
                  type="checkbox"
                  checked={!!form.is_default}
                  onChange={(event) => updateField("is_default", event.target.checked)}
                />
                设为默认
              </label>
            </div>
            <textarea
              value={form.system_prompt || ""}
              onChange={(event) => {
                const value = event.target.value;
                setForm((prev) => ({
                  ...prev,
                  system_prompt: value,
                  system_prompt_masked: false
                }));
              }}
              className="h-28 w-full rounded-2xl border border-black/10 bg-white/80 p-3 text-xs text-ink outline-none focus:border-accent/40"
              placeholder="系统提示词"
            />
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-xs text-muted">Skills</p>
              <Button variant="outline" size="sm" onClick={addSkill}>
                添加技能路径
              </Button>
            </div>
            {(form.skills || []).length === 0 ? (
              <p className="text-xs text-muted">暂无 skills</p>
            ) : (
              <div className="space-y-2">
                {(form.skills || []).map((skill, index) => (
                  <div key={`skill-${index}`} className="flex gap-2">
                    <Input value={skill} onChange={(event) => updateSkill(index, event.target.value)} />
                    <Button variant="outline" size="sm" onClick={() => removeSkill(index)}>
                      删除
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-xs text-muted">MCP Tools</p>
              <Button variant="outline" size="sm" onClick={addTool}>
                添加工具
              </Button>
            </div>
            {(form.allowed_tools || []).length === 0 ? (
              <p className="text-xs text-muted">暂无工具</p>
            ) : (
              <div className="space-y-2">
                {(form.allowed_tools || []).map((tool, index) => (
                  <div key={`tool-${index}`} className="flex gap-2">
                    <Input value={tool} onChange={(event) => updateTool(index, event.target.value)} />
                    <Button variant="outline" size="sm" onClick={() => removeTool(index)}>
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
              <Button variant="outline" size="sm" onClick={addEnvItem}>
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
                        updateEnvItem(index, (prev) => ({ ...prev, key: event.target.value }))
                      }
                      placeholder="KEY"
                    />
                    <Input
                      value={item.value}
                      onChange={(event) =>
                        updateEnvItem(index, (prev) => ({
                          ...prev,
                          value: event.target.value,
                          masked: prev.masked ? false : prev.masked
                        }))
                      }
                      placeholder="VALUE"
                    />
                    <Button variant="outline" size="sm" onClick={() => removeEnvItem(index)}>
                      删除
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="flex flex-wrap gap-3">
            <Button onClick={requestSave} disabled={saving || loading}>
              {saving ? "保存中..." : selectedKey ? "保存修改" : "创建 Profile"}
            </Button>
            {selectedKey ? (
              <Button variant="outline" onClick={requestDelete} disabled={saving}>
                删除 Profile
              </Button>
            ) : null}
          </div>

          {selectedSummary ? (
            <p className="text-xs text-muted">
              当前 profile env 数量：{selectedSummary.env_count}，skills：{selectedSummary.skills_count}，tools：{selectedSummary.tools_count}
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
