"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";

import { clearAdminToken } from "@/lib/admin-token";
import { cn } from "@/lib/cn";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

const navItems = [
  { href: "/", label: "仪表盘", short: "仪" },
  { href: "/release-notes", label: "Release Notes", short: "RN" },
  { href: "/requests", label: "请求实验室", short: "请" },
  { href: "/config", label: "配置管理", short: "配" },
  { href: "/mcp", label: "MCP 管理", short: "MCP" }
];

interface AdminShellProps {
  children: React.ReactNode;
}

export function AdminShell({ children }: AdminShellProps) {
  const pathname = usePathname();
  const router = useRouter();
  const [manualCollapsed, setManualCollapsed] = useState(false);
  const [autoCollapsed, setAutoCollapsed] = useState(false);

  useEffect(() => {
    const handleResize = () => {
      setAutoCollapsed(window.innerWidth < 1280);
    };
    handleResize();
    window.addEventListener("resize", handleResize);
    return () => {
      window.removeEventListener("resize", handleResize);
    };
  }, []);

  const collapsed = autoCollapsed || manualCollapsed;

  const handleLogout = () => {
    clearAdminToken();
    router.replace("/login");
  };

  return (
    <div className="flex min-h-screen">
      <aside
        className={cn(
          "flex-shrink-0 border-r border-black/5 bg-white/60 px-4 py-8 backdrop-blur transition-all",
          collapsed ? "w-16" : "w-56"
        )}
      >
        <div className={cn("flex items-center gap-3", collapsed && "justify-center")}>
          <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-accent text-white shadow-glow">
            GA
          </div>
          {collapsed ? null : (
            <div>
              <p className="text-sm font-semibold tracking-[0.14em] text-ink">GATEWAY</p>
              <p className="text-xs text-muted">Admin control room</p>
            </div>
          )}
        </div>

        <nav className={cn("mt-10 space-y-2", collapsed && "items-center")}>
          {navItems.map((item) => {
            const active = pathname === item.href;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  "flex items-center rounded-2xl px-4 py-3 text-sm font-medium transition",
                  active ? "bg-accent/10 text-accent" : "text-muted hover:bg-black/5",
                  collapsed ? "justify-center" : "justify-between"
                )}
                aria-label={item.label}
              >
                <span className={cn(collapsed ? "sr-only" : "")}>{item.label}</span>
                <span className={cn(collapsed ? "text-xs font-semibold" : "sr-only")}>
                  {item.short}
                </span>
                {active && !collapsed ? <Badge className="border-accent/30">当前</Badge> : null}
              </Link>
            );
          })}
        </nav>

        <div className="mt-10 rounded-2xl border border-black/5 bg-bg-soft/80 p-5">
          <Button
            variant="outline"
            size="sm"
            className={cn("w-full", collapsed && "px-0")}
            onClick={handleLogout}
          >
            {collapsed ? "退出" : "退出登录"}
          </Button>
        </div>
      </aside>

      <div className="flex flex-1 flex-col">
        <header className="flex items-center justify-between border-b border-black/5 bg-white/70 px-6 py-5 backdrop-blur">
          <div className="flex items-center gap-4">
            <Button variant="outline" size="sm" onClick={() => setManualCollapsed((prev) => !prev)}>
              {collapsed ? "展开" : "收起"}
            </Button>
            <div>
              <p className="text-xs uppercase tracking-[0.2em] text-muted">Admin UI</p>
              <h1 className="mt-1 text-xl font-semibold text-ink">CLI Gateway 后台</h1>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <Badge className="bg-accent-2/10 text-accent-2">Live</Badge>
            <Button variant="outline" size="sm" onClick={handleLogout} className="lg:hidden">
              退出登录
            </Button>
          </div>
        </header>
        <main className="flex-1 px-6 py-8">{children}</main>
      </div>
    </div>
  );
}
