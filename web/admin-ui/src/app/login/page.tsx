"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { setAdminToken } from "@/lib/admin-token";

export default function LoginPage() {
  const router = useRouter();
  const [token, setToken] = useState("");
  const [ready, setReady] = useState(false);
  const [nextPath, setNextPath] = useState("/");

  useEffect(() => {
    setReady(true);
    if (typeof window !== "undefined") {
      const params = new URLSearchParams(window.location.search);
      setNextPath(params.get("next") || "/");
    }
  }, []);

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!token.trim()) {
      return;
    }
    setAdminToken(token.trim());
    router.replace(nextPath);
  };

  if (!ready) {
    return null;
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-6 py-16">
      <Card className="w-full max-w-xl p-10 fade-up">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.2em] text-muted">Admin UI</p>
            <h1 className="mt-3 text-3xl font-semibold">输入访问 Token</h1>
            <p className="mt-3 text-sm text-muted">
              后台仅支持 Token 认证，请输入管理员 Token 继续。
            </p>
          </div>
        </div>
        <form className="mt-8 space-y-4" onSubmit={handleSubmit}>
          <Input
            placeholder="请输入 ADMIN_UI_TOKEN"
            value={token}
            onChange={(event) => setToken(event.target.value)}
          />
          <Button type="submit" className="w-full" size="lg">
            进入控制台
          </Button>
        </form>
        <p className="mt-6 text-xs text-muted">
          Token 将保存在浏览器本地存储中，可随时在侧边栏退出登录。
        </p>
      </Card>
    </main>
  );
}
