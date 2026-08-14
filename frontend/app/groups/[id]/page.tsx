"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ArrowLeft, Check, Copy } from "lucide-react";
import { RequireAuth } from "@/components/require-auth";
import { Button } from "@/components/ui/button";
import { ApiError, apiGetAuthed, apiPostAuthed } from "@/lib/api";

type Group = {
  id: string;
  name: string;
  description: string | null;
  group_type: string;
  default_currency: string;
  my_role?: string;
};

function GroupDetail() {
  const params = useParams<{ id: string }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const [group, setGroup] = useState<Group | null>(null);
  const [error, setError] = useState("");
  const [inviteUrl, setInviteUrl] = useState("");
  const [generating, setGenerating] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    apiGetAuthed<{ data: { group: Group } }>(`/groups/${id}`)
      .then((payload) => setGroup(payload.data.group))
      .catch((err) => setError(err instanceof ApiError ? err.message : "Unable to load group"));
  }, [id]);

  async function generateInvite() {
    setGenerating(true);
    setError("");
    try {
      const payload = await apiPostAuthed<{ data: { invite: { invite_url: string } } }>(`/groups/${id}/invites`);
      setInviteUrl(payload.data.invite.invite_url);
      setCopied(false);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to create invite link");
    } finally {
      setGenerating(false);
    }
  }

  async function copyLink() {
    await navigator.clipboard.writeText(inviteUrl);
    setCopied(true);
  }

  return (
    <main className="min-h-screen bg-background px-4 py-6 text-foreground md:px-8">
      <div className="mx-auto max-w-2xl">
        <Link href="/groups" className="mb-4 inline-flex items-center gap-1 text-sm text-muted hover:text-foreground">
          <ArrowLeft className="h-4 w-4" /> Groups
        </Link>

        {error ? <p className="mb-4 rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p> : null}

        {!group ? (
          <p className="text-sm text-muted">Loading…</p>
        ) : (
          <>
            <h1 className="text-2xl font-semibold">{group.name}</h1>
            <p className="mt-1 text-sm capitalize text-muted">
              {group.group_type} · {group.default_currency} · you&apos;re the {group.my_role}
            </p>

            <section className="mt-6 rounded-lg border border-border bg-surface p-4">
              <h2 className="text-base font-semibold">Invite people</h2>
              <p className="mt-1 text-sm text-muted">
                Share this link — anyone who opens it can join the group after signing in.
              </p>
              {inviteUrl ? (
                <div className="mt-3 flex items-center gap-2">
                  <input
                    readOnly
                    value={inviteUrl}
                    className="h-11 flex-1 truncate rounded-lg border border-border bg-background px-3 text-sm text-foreground"
                  />
                  <Button variant="secondary" size="icon" onClick={copyLink} aria-label="Copy link" title="Copy link">
                    {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                  </Button>
                </div>
              ) : (
                <Button className="mt-3" onClick={generateInvite} disabled={generating}>
                  {generating ? "Generating…" : "Create invite link"}
                </Button>
              )}
              <p className="mt-2 text-xs text-muted">Link expires in 7 days.</p>
            </section>

            <section className="mt-4 rounded-lg border border-dashed border-border p-4 text-center text-sm text-muted">
              Expenses and balances for this group are coming soon.
            </section>
          </>
        )}
      </div>
    </main>
  );
}

export default function GroupDetailPage() {
  return (
    <RequireAuth>
      <GroupDetail />
    </RequireAuth>
  );
}
