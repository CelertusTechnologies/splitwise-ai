"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { AuthPanel } from "@/components/auth-panel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiPost } from "@/lib/api";

export default function ResetPasswordPage() {
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
      await apiPost("/auth/reset-password", {
        body: {
          token: form.get("token"),
          new_password: form.get("new_password")
        }
      });
      setMessage("Password reset successfully.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to reset password");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthPanel
      title="Set new password"
      subtitle="Use the reset token and choose a stronger password."
      footer={
        <Link className="font-medium text-primary" href="/login">
          Sign in
        </Link>
      }
    >
      <form className="grid gap-4" onSubmit={handleSubmit}>
        <Input label="Reset token" name="token" type="text" required />
        <Input label="New password" name="new_password" type="password" autoComplete="new-password" required minLength={12} />
        {message ? <p className="rounded-lg bg-primary/10 px-3 py-2 text-sm text-primary">{message}</p> : null}
        {error ? <p className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p> : null}
        <Button type="submit" disabled={loading}>
          {loading ? "Updating" : "Update password"}
        </Button>
      </form>
    </AuthPanel>
  );
}

