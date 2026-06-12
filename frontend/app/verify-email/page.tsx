"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { AuthPanel } from "@/components/auth-panel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiPost } from "@/lib/api";

export default function VerifyEmailPage() {
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setMessage("");
    setLoading(true);

    const form = new FormData(event.currentTarget);
    try {
      await apiPost("/auth/verify-email", {
        body: { token: form.get("token") }
      });
      setMessage("Email verified successfully.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to verify email");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthPanel
      title="Verify email"
      subtitle="Confirm ownership before inviting groups and recording settlements."
      footer={
        <Link className="font-medium text-primary" href="/dashboard">
          Open dashboard
        </Link>
      }
    >
      <form className="grid gap-4" onSubmit={handleSubmit}>
        <Input label="Verification token" name="token" type="text" required />
        {message ? <p className="rounded-lg bg-primary/10 px-3 py-2 text-sm text-primary">{message}</p> : null}
        {error ? <p className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p> : null}
        <Button type="submit" disabled={loading}>
          {loading ? "Verifying" : "Verify email"}
        </Button>
      </form>
    </AuthPanel>
  );
}

