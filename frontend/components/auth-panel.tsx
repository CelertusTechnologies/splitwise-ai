import Link from "next/link";
import { ThemeToggle } from "@/components/ui/theme-toggle";

type AuthPanelProps = {
  title: string;
  subtitle: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
};

export function AuthPanel({ title, subtitle, children, footer }: AuthPanelProps) {
  return (
    <main className="grid min-h-screen bg-background px-4 py-6 text-foreground md:place-items-center">
      <section className="w-full max-w-md rounded-lg border border-border bg-surface p-5 shadow-panel md:p-6">
        <div className="mb-8 flex items-center justify-between gap-3">
          <Link href="/" className="flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-foreground font-semibold text-background">
              N
            </span>
            <span className="text-sm font-semibold">Nivra</span>
          </Link>
          <ThemeToggle />
        </div>
        <div className="mb-6">
          <h1 className="text-2xl font-semibold">{title}</h1>
          <p className="mt-2 text-sm leading-6 text-muted">{subtitle}</p>
        </div>
        {children}
        <div className="mt-6 text-center text-sm text-muted">{footer}</div>
      </section>
    </main>
  );
}

