import {
  ArrowDownRight,
  ArrowUpRight,
  Bell,
  CheckCircle2,
  CircleDollarSign,
  CreditCard,
  IndianRupee,
  LayoutDashboard,
  Plus,
  ReceiptText,
  Search,
  Settings,
  TrendingUp,
  UsersRound,
  WalletCards
} from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import { formatINR } from "@/lib/utils";

const metrics = [
  {
    label: "Total owed",
    value: 18420,
    helper: "Across 6 groups",
    tone: "text-primary",
    icon: ArrowDownRight
  },
  {
    label: "You owe",
    value: 7380,
    helper: "3 settlements pending",
    tone: "text-danger",
    icon: ArrowUpRight
  },
  {
    label: "Monthly spend",
    value: 54260,
    helper: "12% below May",
    tone: "text-info",
    icon: TrendingUp
  },
  {
    label: "Active groups",
    value: 8,
    helper: "Trip, home, friends",
    tone: "text-accent",
    icon: UsersRound
  }
];

const groups = [
  { name: "Goa Long Weekend", type: "Trip", balance: 12600, members: 7, color: "bg-primary" },
  { name: "Indiranagar Flat", type: "Flatmates", balance: -3180, members: 4, color: "bg-info" },
  { name: "Family Monthly", type: "Family", balance: 4400, members: 5, color: "bg-accent" }
];

const activity = [
  { title: "Aarav added dinner at Vinayak", meta: "Goa Long Weekend", amount: -1240 },
  { title: "Meera settled via UPI", meta: "Family Monthly", amount: 2200 },
  { title: "You uploaded a fuel receipt", meta: "Indiranagar Flat", amount: -980 }
];

const bars = [42, 68, 35, 78, 54, 88, 64, 72, 48, 82, 58, 91];

export function DashboardShell() {
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
          {[
            { label: "Dashboard", icon: LayoutDashboard },
            { label: "Groups", icon: UsersRound },
            { label: "Expenses", icon: ReceiptText },
            { label: "Settlements", icon: WalletCards },
            { label: "Settings", icon: Settings }
          ].map((item) => (
            <Link
              key={item.label}
              href="/dashboard"
              className="flex h-10 items-center gap-3 rounded-lg px-3 text-muted transition hover:bg-foreground/5 hover:text-foreground"
            >
              <item.icon aria-hidden className="h-4 w-4" />
              {item.label}
            </Link>
          ))}
        </nav>

        <div className="mt-auto rounded-lg border border-border bg-background p-4">
          <div className="mb-3 flex items-center gap-2 text-sm font-medium">
            <CheckCircle2 className="h-4 w-4 text-primary" />
            Settlement health
          </div>
          <div className="h-2 rounded-full bg-foreground/10">
            <div className="h-2 w-3/4 rounded-full bg-primary" />
          </div>
          <p className="mt-3 text-xs leading-5 text-muted">74% of balances are already settled this month.</p>
        </div>
      </aside>

      <section className="lg:pl-72">
        <header className="sticky top-0 z-10 border-b border-border bg-background/90 px-4 py-3 backdrop-blur md:px-6">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-xs font-medium uppercase text-muted">Tuesday, 9 Jun</p>
              <h1 className="text-xl font-semibold md:text-2xl">Dashboard</h1>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="secondary" size="icon" aria-label="Search" title="Search">
                <Search aria-hidden className="h-4 w-4" />
              </Button>
              <Button variant="secondary" size="icon" aria-label="Notifications" title="Notifications">
                <Bell aria-hidden className="h-4 w-4" />
              </Button>
              <ThemeToggle />
              <Button className="hidden md:inline-flex">
                <Plus aria-hidden className="h-4 w-4" />
                Add expense
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
              <Button variant="secondary" size="sm">
                <IndianRupee aria-hidden className="h-4 w-4" />
                INR
              </Button>
            </div>
            <div className="flex h-64 items-end gap-2">
              {bars.map((height, index) => (
                <div key={index} className="flex flex-1 items-end">
                  <div
                    className="w-full rounded-t-md bg-info/70"
                    style={{ height: `${height}%` }}
                    aria-label={`Week ${index + 1}`}
                  />
                </div>
              ))}
            </div>
          </section>

          <section className="rounded-lg border border-border bg-surface p-4 shadow-panel lg:col-span-5">
            <div className="mb-4 flex items-center justify-between gap-3">
              <div>
                <h2 className="text-base font-semibold">Active groups</h2>
                <p className="text-sm text-muted">Balances that changed recently</p>
              </div>
              <Button variant="ghost" size="sm">View all</Button>
            </div>
            <div className="grid gap-3">
              {groups.map((group) => (
                <div key={group.name} className="flex items-center justify-between gap-3 border-b border-border pb-3 last:border-0 last:pb-0">
                  <div className="flex min-w-0 items-center gap-3">
                    <span className={`h-10 w-10 shrink-0 rounded-lg ${group.color}`} />
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-medium">{group.name}</span>
                      <span className="block text-xs text-muted">{group.type} · {group.members} members</span>
                    </span>
                  </div>
                  <span className={group.balance >= 0 ? "text-sm font-semibold text-primary" : "text-sm font-semibold text-danger"}>
                    {formatINR(Math.abs(group.balance))}
                  </span>
                </div>
              ))}
            </div>
          </section>

          <section className="rounded-lg border border-border bg-surface p-4 shadow-panel lg:col-span-6">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-base font-semibold">Settlement plan</h2>
              <CircleDollarSign className="h-5 w-5 text-primary" />
            </div>
            <div className="grid gap-3">
              {[
                "You pay Rohan INR 2,400",
                "Kavya pays you INR 3,100",
                "Anika pays Meera INR 1,250"
              ].map((item) => (
                <div key={item} className="flex items-center gap-3 rounded-lg bg-background px-3 py-3 text-sm">
                  <CheckCircle2 className="h-4 w-4 text-primary" />
                  {item}
                </div>
              ))}
            </div>
          </section>

          <section className="rounded-lg border border-border bg-surface p-4 shadow-panel lg:col-span-6">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-base font-semibold">Activity</h2>
              <CreditCard className="h-5 w-5 text-accent" />
            </div>
            <div className="grid gap-3">
              {activity.map((item) => (
                <div key={item.title} className="flex items-center justify-between gap-3 border-b border-border pb-3 last:border-0 last:pb-0">
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-medium">{item.title}</span>
                    <span className="block text-xs text-muted">{item.meta}</span>
                  </span>
                  <span className={item.amount >= 0 ? "text-sm font-semibold text-primary" : "text-sm font-semibold text-danger"}>
                    {item.amount >= 0 ? "+" : "-"}{formatINR(Math.abs(item.amount))}
                  </span>
                </div>
              ))}
            </div>
          </section>
        </div>
      </section>

      <nav className="fixed inset-x-0 bottom-0 grid grid-cols-4 border-t border-border bg-surface px-2 py-2 lg:hidden">
        {[
          { label: "Home", icon: LayoutDashboard },
          { label: "Groups", icon: UsersRound },
          { label: "Add", icon: Plus },
          { label: "Settle", icon: WalletCards }
        ].map((item) => (
          <Link key={item.label} href="/dashboard" className="flex flex-col items-center gap-1 rounded-lg px-2 py-2 text-xs text-muted">
            <item.icon aria-hidden className="h-5 w-5" />
            {item.label}
          </Link>
        ))}
      </nav>
    </main>
  );
}

