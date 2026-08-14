"use client";

import { FormEvent, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError, apiPost, getPendingInvite, setTokens, type Tokens } from "@/lib/api";

type RequestOTPResponse = { data: { dev_otp?: string; message: string } };

type VerifyOTPResponse = {
  data:
    | { is_new_user: true }
    | { is_new_user: false; user: { full_name: string }; tokens: Tokens };
};

type CompleteSignupResponse = {
  data: { user: { full_name: string }; tokens: Tokens };
};

type Step = "phone" | "code" | "details";

function redirectAfterAuth() {
  const pendingInvite = getPendingInvite();
  window.location.assign(pendingInvite ? `/join/${pendingInvite}` : "/dashboard");
}

export function OTPLoginForm() {
  const [step, setStep] = useState<Step>("phone");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [devOtp, setDevOtp] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function requestCode(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);
    const value = String(new FormData(event.currentTarget).get("phone_number") ?? "").trim();
    try {
      const payload = await apiPost<RequestOTPResponse>("/auth/otp/request", {
        body: { phone_number: value }
      });
      setPhone(value);
      setDevOtp(payload.data.dev_otp ?? "");
      setStep("code");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to send code");
    } finally {
      setLoading(false);
    }
  }

  async function verifyCode(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);
    const value = String(new FormData(event.currentTarget).get("code") ?? "").trim();
    try {
      const payload = await apiPost<VerifyOTPResponse>("/auth/otp/verify", {
        body: { phone_number: phone, code: value }
      });
      setCode(value);
      if (payload.data.is_new_user) {
        setStep("details");
      } else {
        setTokens(payload.data.tokens);
        redirectAfterAuth();
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Invalid code");
    } finally {
      setLoading(false);
    }
  }

  async function completeSignup(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);
    const form = new FormData(event.currentTarget);
    try {
      const payload = await apiPost<CompleteSignupResponse>("/auth/otp/complete-signup", {
        body: {
          phone_number: phone,
          code,
          full_name: form.get("full_name"),
          email: form.get("email")
        }
      });
      setTokens(payload.data.tokens);
      redirectAfterAuth();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to finish creating your account");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="grid gap-4">
      {error ? <p className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p> : null}

      {step === "phone" ? (
        <form className="grid gap-4" onSubmit={requestCode}>
          <Input
            label="Phone number"
            name="phone_number"
            type="tel"
            autoComplete="tel"
            required
            minLength={8}
            placeholder="+91 98765 43210"
          />
          <Button type="submit" disabled={loading}>
            {loading ? "Sending…" : "Send code"}
          </Button>
        </form>
      ) : null}

      {step === "code" ? (
        <form className="grid gap-4" onSubmit={verifyCode}>
          <p className="text-sm text-muted">Enter the 6-digit code sent to {phone}.</p>
          {devOtp ? (
            <p className="rounded-lg border border-primary/30 bg-primary/10 px-3 py-2 text-xs text-primary">
              No SMS provider is connected yet — your code is <code className="font-semibold">{devOtp}</code>
            </p>
          ) : null}
          <Input
            label="Code"
            name="code"
            type="text"
            inputMode="numeric"
            autoComplete="one-time-code"
            required
            minLength={6}
            maxLength={6}
          />
          <Button type="submit" disabled={loading}>
            {loading ? "Verifying…" : "Verify"}
          </Button>
          <button
            type="button"
            className="text-sm text-muted underline-offset-2 hover:underline"
            onClick={() => {
              setStep("phone");
              setError("");
            }}
          >
            Use a different number
          </button>
        </form>
      ) : null}

      {step === "details" ? (
        <form className="grid gap-4" onSubmit={completeSignup}>
          <p className="text-sm text-muted">
            This number is new to Nivra — tell us a little about you to finish setting up your account.
          </p>
          <Input label="Full name" name="full_name" type="text" autoComplete="name" required minLength={2} />
          <Input label="Email" name="email" type="email" autoComplete="email" required />
          <Button type="submit" disabled={loading}>
            {loading ? "Finishing…" : "Create account"}
          </Button>
        </form>
      ) : null}
    </div>
  );
}
