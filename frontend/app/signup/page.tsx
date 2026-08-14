"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { AuthPanel } from "@/components/auth-panel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiPost, setTokens } from "@/lib/api";

type SignUpResponse = {
  data: {
    dev_email_verification_token?: string;
    tokens: {
      access_token: string;
      refresh_token: string;
    };
  };
};

export default function SignUpPage() {
  const [error, setError] = useState("");
  const [devToken, setDevToken] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setDevToken("");
    setLoading(true);

    const form = new FormData(event.currentTarget);
    try {
      const payload = await apiPost<SignUpResponse>("/auth/signup", {
        body: {
          full_name: form.get("full_name"),
          email: form.get("email"),
          phone_number: form.get("phone_number") || undefined,
          password: form.get("password"),
          preferred_currency: "INR"
        }
      });
      setTokens(payload.data.tokens);
      if (payload.data.dev_email_verification_token) {
        setDevToken(payload.data.dev_email_verification_token);
      } else {
        window.location.assign("/dashboard");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to create account");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthPanel
      title="Create account"
      subtitle="Start with INR balances, private groups, and token-secured access."
      footer={
        <>
          Already registered?{" "}
          <Link className="font-medium text-primary" href="/login">
            Sign in
          </Link>
        </>
      }
    >
      <form className="grid gap-4" onSubmit={handleSubmit}>
        <Input label="Full name" name="full_name" type="text" autoComplete="name" required minLength={2} />
        <Input label="Email" name="email" type="email" autoComplete="email" required />
        <Input label="Phone number" name="phone_number" type="tel" autoComplete="tel" />
        <Input label="Password" name="password" type="password" autoComplete="new-password" required minLength={12} />
        {error ? <p className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p> : null}
        {devToken ? (
          <div className="rounded-lg border border-primary/30 bg-primary/10 p-3 text-sm">
            <p className="font-medium text-primary">Development verification token</p>
            <code className="mt-2 block break-all text-xs text-foreground">{devToken}</code>
            <Link className="mt-3 inline-block font-medium text-primary" href="/verify-email">
              Verify email
            </Link>
          </div>
        ) : null}
        <Button type="submit" disabled={loading}>
          {loading ? "Creating account" : "Create account"}
        </Button>
      </form>
    </AuthPanel>
  );
}

