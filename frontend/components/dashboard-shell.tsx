"use client";

import {
  ArrowDownRight,
  ArrowUpRight,
  Bell,
  CheckCircle2,
  LayoutDashboard,
  LogOut,
  Plus,
  ReceiptText,
  Settings,
  TrendingUp,
  UsersRound,
  WalletCards
} from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import { formatINR } from "@/lib/utils";
import { useAuth } from "@/lib/auth-context";

const navItems = [
  { label: "Dashboard", icon: LayoutDashboard, available: true },
  { label: "Groups", icon: UsersRound, available: false },
  { label: "Expenses", icon: ReceiptText, available: false },
  { label: "Settlements", icon: WalletCards, available: false },
  { label: "Settings", icon: Settings, available: false }
];

const metrics = [
  { label: "Total owed", value: 0, helper: "Across 0 groups", tone: "text-primary", icon: ArrowDownRight },
  { label: "You owe", value: 0, helper: "No settlements pending", tone: "text-danger", icon: ArrowUpRight },
  { label: "Monthly spend", value: 0, helper: "No expenses logged yet", tone: "text-info", icon: TrendingUp },
  { label: "Active groups", value: 0, helper: "Create your first group", tone: "text-accent", icon: UsersRound }
];

export function DashboardShell() {
  const { user, logout } = useAuth();
  const today = new Date().toLocaleDateString("en-IN", { weekday: "long", day: "numeric", month: "short" });

  return (
    <main className="min-h-screen bg-background text-foreground">
      <aside className="fixed inset-y-0 left-0 hidden w-72 border-r border-border bg-surface px-4 py-5 lg:flex lg:flex-col">
        <Link href="/dashboard" className="mb-8 flex items-center gap-3">
          <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-foreground text-base font-semibold text-background">
            N
          </span>
          <span>
            <span className="block text-sm font-semibold">Nivra</span>
            <span className="block text-xs text-muted">Shared money</span>
          </span>
        </Link>

        <nav className="grid gap-1 text-sm">
          {navItems.map((item) => (
            <Link
              key={item.label}
              href="/dashboard"
              aria-disabled={!item.available}
              onClick={(event) => {
                if (!item.available) event.preventDefault();
              }}
              className={
                item.available
                  ? "flex h-10 items-center gap-3 rounded-lg px-3 text-muted transition hover:bg-foreground/5 hover:text-foreground"
                  : "flex h-10 items-center justify-between gap-3 rounded-lg px-3 text-muted/50 cursor-default"
              }
            >
              <span className="flex items-center gap-3">
                <item.icon aria-hidden className="h-4 w-4" />
                {item.label}
              </span>
              {!item.available ? <span className="text-[10px] uppercase tracking-wide text-muted/60">Soon</span> : null}
            </Link>
          ))}
        </nav>

        <div className="mt-auto grid gap-2">
          <div className="rounded-lg border border-border bg-background p-4">
            <p className="truncate text-sm font-medium">{user?.full_name}</p>
            <p className="truncate text-xs text-muted">{user?.email}</p>
          </div>
          <Button variant="secondary" size="sm" onClick={logout} className="justify-center">
            <LogOut aria-hidden className="h-4 w-4" />
            Log out
          </Button>
        </div>
      </aside>

      <section className="lg:pl-72">
        <header className="sticky top-0 z-10 border-b border-border bg-background/90 px-4 py-3 backdrop-blur md:px-6">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-xs font-medium uppercase text-muted">{today}</p>
              <h1 className="text-xl font-semibold md:text-2xl">Welcome, {user?.full_name?.split(" ")[0] ?? "there"}</h1>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="secondary" size="icon" aria-label="Notifications" title="No notifications yet" disabled>
                <Bell aria-hidden className="h-4 w-4" />
              </Button>
              <ThemeToggle />
              <Button className="hidden md:inline-flex" disabled title="Group and expense creation is coming soon">
                <Plus aria-hidden className="h-4 w-4" />
                Add expense
              </Button>
              <Button variant="secondary" size="icon" aria-label="Log out" title="Log out" onClick={logout} className="lg:hidden">
                <LogOut aria-hidden className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </header>

        <div className="grid gap-4 px-4 py-4 pb-24 md:px-6 lg:grid-cols-12">
          <section className="grid gap-4 sm:grid-cols-2 lg:col-span-12 xl:grid-cols-4">
            {metrics.map((metric) => (
              <article key={metric.label} className="rounded-lg border border-border bg-surface p-4 shadow-panel">
                <div className="mb-5 flex items-center justify-between gap-3">
                  <span className="text-sm text-muted">{metric.label}</span>
                  <metric.icon aria-hidden className={`h-5 w-5 ${metric.tone}`} />
                </div>
                <strong className="block text-2xl font-semibold">
                  {metric.label === "Active groups" ? metric.value : formatINR(metric.value)}
                </strong>
                <span className="mt-1 block text-sm text-muted">{metric.helper}</span>
              </article>
            ))}
          </section>

          <section className="rounded-lg border border-border bg-surface p-4 shadow-panel lg:col-span-7">
            <div className="mb-6 flex items-center justify-between gap-3">
              <div>
                <h2 className="text-base font-semibold">Spending trend</h2>
                <p className="text-sm text-muted">Last 12 weeks</p>
              </div>
            </div>
            <div className="flex h-48 items-center justify-center rounded-lg border border-dashed border-border text-sm text-muted">
              No expenses logged yet — trends will show up here once you add some.
            </div>
          </section>

          <section className="rounded-lg border border-border bg-surface p-4 shadow-panel lg:col-span-5">
            <div className="mb-4 flex items-center justify-between gap-3">
              <div>
                <h2 className="text-base font-semibold">Active groups</h2>
                <p className="text-sm text-muted">Balances that changed recently</p>
              </div>
            </div>
            <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed border-border px-4 py-8 text-center">
              <UsersRound aria-hidden className="h-6 w-6 text-muted" />
              <p className="text-sm text-muted">You're not in any groups yet.</p>
              <Button variant="secondary" size="sm" disabled title="Group creation is coming soon">
                Create a group
              </Button>
            </div>
          </section>

          <section className="rounded-lg border border-border bg-surface p-4 shadow-panel lg:col-span-6">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-base font-semibold">Settlement plan</h2>
            </div>
            <div className="flex items-center gap-3 rounded-lg bg-background px-3 py-6 text-sm text-muted">
              <CheckCircle2 className="h-4 w-4 text-primary" />
              You're all settled up — nothing to pay or collect.
            </div>
          </section>

          <section className="rounded-lg border border-border bg-surface p-4 shadow-panel lg:col-span-6">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-base font-semibold">Activity</h2>
            </div>
            <div className="flex items-center rounded-lg bg-background px-3 py-6 text-sm text-muted">
              No activity yet. Once you add expenses or settle up, it'll show here.
            </div>
          </section>
        </div>
      </section>

      <nav className="fixed inset-x-0 bottom-0 grid grid-cols-4 border-t border-border bg-surface px-2 py-2 lg:hidden">
        {[
          { label: "Home", icon: LayoutDashboard, available: true },
          { label: "Groups", icon: UsersRound, available: false },
          { label: "Add", icon: Plus, available: false },
          { label: "Settle", icon: WalletCards, available: false }
        ].map((item) => (
          <Link
            key={item.label}
            href="/dashboard"
            aria-disabled={!item.available}
            onClick={(event) => {
              if (!item.available) event.preventDefault();
            }}
            className={`flex flex-col items-center gap-1 rounded-lg px-2 py-2 text-xs ${
              item.available ? "text-muted" : "text-muted/40"
            }`}
          >
            <item.icon aria-hidden className="h-5 w-5" />
            {item.label}
          </Link>
        ))}
      </nav>
    </main>
  );
}
