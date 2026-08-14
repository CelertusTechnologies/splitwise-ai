"use client";

import Link from "next/link";
import { FormEvent, useState } from "react";
import { AuthPanel } from "@/components/auth-panel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiPost, setTokens } from "@/lib/api";

type AuthResponse = {
  data: {
    tokens: {
      access_token: string;
      refresh_token: string;
    };
    user: {
      full_name: string;
    };
  };
};

export default function LoginPage() {
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);

    const form = new FormData(event.currentTarget);
    try {
      const payload = await apiPost<AuthResponse>("/auth/login", {
        body: {
          email: form.get("email"),
          password: form.get("password")
        }
      });
      setTokens(payload.data.tokens);
      window.location.assign("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to sign in");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthPanel
      title="Welcome back"
      subtitle="Open your balances, expenses, and settlement plan."
      footer={
        <>
          New to Nivra?{" "}
          <Link className="font-medium text-primary" href="/signup">
            Create account
          </Link>
        </>
      }
    >
      <form className="grid gap-4" onSubmit={handleSubmit}>
        <Input label="Email" name="email" type="email" autoComplete="email" required />
        <Input label="Password" name="password" type="password" autoComplete="current-password" required minLength={12} />
        <div className="flex items-center justify-between gap-3">
          <Link className="text-sm font-medium text-primary" href="/forgot-password">
            Forgot password
          </Link>
        </div>
        {error ? <p className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p> : null}
        <Button type="submit" disabled={loading}>
          {loading ? "Signing in" : "Sign in"}
        </Button>
      </form>
    </AuthPanel>
  );
}

