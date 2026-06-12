"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { AuthPanel } from "@/components/auth-panel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiPost } from "@/lib/api";

type ForgotResponse = {
  data: {
    message: string;
    dev_reset_token?: string;
  };
};

export default function ForgotPasswordPage() {
  const [message, setMessage] = useState("");
  const [devToken, setDevToken] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setMessage("");
    setDevToken("");
    setLoading(true);

    const form = new FormData(event.currentTarget);
    try {
      const payload = await apiPost<ForgotResponse>("/auth/forgot-password", {
        body: { email: form.get("email") }
      });
      setMessage(payload.data.message);
      setDevToken(payload.data.dev_reset_token ?? "");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to request reset");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthPanel
      title="Reset password"
      subtitle="Request a one-time reset token for your account email."
      footer={
        <Link className="font-medium text-primary" href="/login">
          Return to sign in
        </Link>
      }
    >
      <form className="grid gap-4" onSubmit={handleSubmit}>
        <Input label="Email" name="email" type="email" autoComplete="email" required />
        {message ? <p className="rounded-lg bg-primary/10 px-3 py-2 text-sm text-primary">{message}</p> : null}
        {devToken ? (
          <div className="rounded-lg border border-primary/30 bg-primary/10 p-3 text-sm">
            <p className="font-medium text-primary">Development reset token</p>
            <code className="mt-2 block break-all text-xs text-foreground">{devToken}</code>
            <Link className="mt-3 inline-block font-medium text-primary" href="/reset-password">
              Continue reset
            </Link>
          </div>
        ) : null}
        {error ? <p className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p> : null}
        <Button type="submit" disabled={loading}>
          {loading ? "Sending" : "Send reset link"}
        </Button>
      </form>
    </AuthPanel>
  );
}

