"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, Plus, UsersRound } from "lucide-react";
import { RequireAuth } from "@/components/require-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError, apiGetAuthed, apiPostAuthed } from "@/lib/api";

type Group = {
  id: string;
  name: string;
  description: string | null;
  group_type: string;
  default_currency: string;
  status: string;
};

function GroupsList() {
  const [groups, setGroups] = useState<Group[] | null>(null);
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [creating, setCreating] = useState(false);

  async function load() {
    try {
      const payload = await apiGetAuthed<{ data: { groups: Group[] } }>("/groups");
      setGroups(payload.data.groups);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to load groups");
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreating(true);
    setError("");
    const form = event.currentTarget;
    const data = new FormData(form);
    try {
      await apiPostAuthed("/groups", {
        name: data.get("name"),
        group_type: data.get("group_type") || undefined
      });
      setShowForm(false);
      form.reset();
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to create group");
    } finally {
      setCreating(false);
    }
  }

  return (
    <main className="min-h-screen bg-background px-4 py-6 text-foreground md:px-8">
      <div className="mx-auto max-w-3xl">
        <div className="mb-6 flex items-start justify-between gap-3">
          <div>
            <Link href="/dashboard" className="mb-2 inline-flex items-center gap-1 text-sm text-muted hover:text-foreground">
              <ArrowLeft className="h-4 w-4" /> Dashboard
            </Link>
            <h1 className="text-2xl font-semibold">Groups</h1>
          </div>
          <Button onClick={() => setShowForm((value) => !value)}>
            <Plus className="h-4 w-4" />
            New group
          </Button>
        </div>

        {showForm ? (
          <form onSubmit={handleCreate} className="mb-6 grid gap-4 rounded-lg border border-border bg-surface p-4">
            <Input label="Group name" name="name" required minLength={2} maxLength={120} placeholder="Goa Trip" />
            <label className="grid gap-2 text-sm font-medium text-foreground">
              <span>Type</span>
              <select
                name="group_type"
                defaultValue="custom"
                className="h-11 rounded-lg border border-border bg-surface px-3 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
              >
                <option value="custom">Custom</option>
                <option value="trip">Trip</option>
                <option value="family">Family</option>
                <option value="friends">Friends</option>
                <option value="flatmates">Flatmates</option>
                <option value="couple">Couple</option>
              </select>
            </label>
            <Button type="submit" disabled={creating}>
              {creating ? "Creating…" : "Create group"}
            </Button>
          </form>
        ) : null}

        {error ? <p className="mb-4 rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p> : null}

        {groups === null ? (
          <p className="text-sm text-muted">Loading…</p>
        ) : groups.length === 0 ? (
          <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed border-border px-4 py-12 text-center">
            <UsersRound aria-hidden className="h-6 w-6 text-muted" />
            <p className="text-sm text-muted">You&apos;re not in any groups yet. Create one to get started.</p>
          </div>
        ) : (
          <div className="grid gap-3">
            {groups.map((g) => (
              <Link
                key={g.id}
                href={`/groups/${g.id}`}
                className="flex items-center justify-between gap-3 rounded-lg border border-border bg-surface p-4 transition hover:bg-foreground/5"
              >
                <div className="min-w-0">
                  <p className="truncate font-medium">{g.name}</p>
                  <p className="text-xs capitalize text-muted">
                    {g.group_type} · {g.default_currency}
                  </p>
                </div>
              </Link>
            ))}
          </div>
        )}
      </div>
    </main>
  );
}

export default function GroupsPage() {
  return (
    <RequireAuth>
      <GroupsList />
    </RequireAuth>
  );
}
