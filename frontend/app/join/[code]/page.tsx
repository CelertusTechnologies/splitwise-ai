"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { ApiError, apiPostAuthed, clearPendingInvite, setPendingInvite } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";

type JoinResponse = {
  data: {
    group: {
      id: string;
      name: string;
    };
  };
};

export default function JoinGroupPage() {
  const params = useParams<{ code: string }>();
  const code = Array.isArray(params.code) ? params.code[0] : params.code;
  const { user, loading } = useAuth();
  const router = useRouter();
  const [status, setStatus] = useState<"working" | "error">("working");
  const [error, setError] = useState("");

  useEffect(() => {
    if (loading) return;

    if (!user) {
      setPendingInvite(code);
      router.replace("/login");
      return;
    }

    let cancelled = false;
    async function join() {
      try {
        const payload = await apiPostAuthed<JoinResponse>("/groups/join", { invite_code: code });
        clearPendingInvite();
        if (!cancelled) router.replace(`/groups/${payload.data.group.id}`);
      } catch (err) {
        if (cancelled) return;
        setStatus("error");
        setError(err instanceof ApiError ? err.message : "Unable to join this group");
      }
    }
    join();

    return () => {
      cancelled = true;
    };
  }, [loading, user, code, router]);

  return (
    <main className="flex min-h-screen items-center justify-center bg-background px-4 text-center text-foreground">
      <div className="max-w-sm">
        {status === "error" ? (
          <>
            <p className="text-lg font-semibold">Couldn't join group</p>
            <p className="mt-2 text-sm text-muted">{error}</p>
            <Button asChild className="mt-4">
              <Link href="/dashboard">Back to dashboard</Link>
            </Button>
          </>
        ) : (
          <p className="text-sm text-muted">Joining group…</p>
        )}
      </div>
    </main>
  );
}
