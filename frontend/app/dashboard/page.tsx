import { DashboardShell } from "@/components/dashboard-shell";
import { RequireAuth } from "@/components/require-auth";

export default function DashboardPage() {
  return (
    <RequireAuth>
      <DashboardShell />
    </RequireAuth>
  );
}
