"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ArrowLeft, Check, Copy, Plus, Receipt, Trash2 } from "lucide-react";
import { RequireAuth } from "@/components/require-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/lib/auth-context";
import { ApiError, apiDeleteAuthed, apiGetAuthed, apiPostAuthed } from "@/lib/api";
import { formatINR } from "@/lib/utils";

type Group = {
  id: string;
  name: string;
  description: string | null;
  group_type: string;
  default_currency: string;
  my_role?: string;
};

type Member = {
  user_id: string;
  full_name: string;
  email: string;
  role: string;
};

type Category = {
  slug: string;
  name: string;
  icon: string;
};

type Expense = {
  id: string;
  title: string;
  amount: number;
  currency: string;
  category_slug: string | null;
  split_method: string;
  paid_by_user_id: string;
  expense_date: string;
};

type Balance = {
  user_id: string;
  net: number;
};

type SplitMethod = "equal" | "unequal" | "percentage" | "shares";

type ParticipantState = {
  selected: boolean;
  amount: string;
  percentage: string;
  shares: string;
};

function GroupDetail() {
  const params = useParams<{ id: string }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { user } = useAuth();

  const [group, setGroup] = useState<Group | null>(null);
  const [members, setMembers] = useState<Member[] | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [expenses, setExpenses] = useState<Expense[] | null>(null);
  const [balances, setBalances] = useState<Balance[] | null>(null);

  const [error, setError] = useState("");
  const [inviteUrl, setInviteUrl] = useState("");
  const [generating, setGenerating] = useState(false);
  const [copied, setCopied] = useState(false);
  const [showExpenseForm, setShowExpenseForm] = useState(false);

  useEffect(() => {
    apiGetAuthed<{ data: { group: Group } }>(`/groups/${id}`)
      .then((payload) => setGroup(payload.data.group))
      .catch((err) => setError(err instanceof ApiError ? err.message : "Unable to load group"));

    apiGetAuthed<{ data: { members: Member[] } }>(`/groups/${id}/members`)
      .then((payload) => setMembers(payload.data.members))
      .catch(() => setMembers([]));

    apiGetAuthed<{ data: { categories: Category[] } }>("/expense-categories")
      .then((payload) => setCategories(payload.data.categories))
      .catch(() => setCategories([]));
  }, [id]);

  async function loadExpenses() {
    try {
      const [expensesPayload, balancesPayload] = await Promise.all([
        apiGetAuthed<{ data: { expenses: Expense[] } }>(`/groups/${id}/expenses`),
        apiGetAuthed<{ data: { balances: Balance[] } }>(`/groups/${id}/balances`)
      ]);
      setExpenses(expensesPayload.data.expenses);
      setBalances(balancesPayload.data.balances);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to load expenses");
    }
  }

  useEffect(() => {
    loadExpenses();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  const memberName = useMemo(() => {
    const byID = new Map((members ?? []).map((m) => [m.user_id, m.full_name]));
    return (userID: string) => byID.get(userID) ?? "Someone";
  }, [members]);

  const categoryName = useMemo(() => {
    const bySlug = new Map(categories.map((c) => [c.slug, c.name]));
    return (slug: string | null) => (slug ? bySlug.get(slug) ?? slug : null);
  }, [categories]);

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

  async function handleDeleteExpense(expenseID: string) {
    if (!window.confirm("Delete this expense? This can't be undone.")) return;
    setError("");
    try {
      await apiDeleteAuthed(`/expenses/${expenseID}`);
      await loadExpenses();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to delete expense");
    }
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

            <section className="mt-4 rounded-lg border border-border bg-surface p-4">
              <h2 className="text-base font-semibold">Balances</h2>
              {balances === null ? (
                <p className="mt-3 text-sm text-muted">Loading…</p>
              ) : balances.length === 0 ? (
                <p className="mt-3 text-sm text-muted">No expenses yet — balances will show up once you add one.</p>
              ) : (
                <div className="mt-3 grid gap-2">
                  {balances.map((b) => (
                    <div key={b.user_id} className="flex items-center justify-between text-sm">
                      <span>{memberName(b.user_id)}</span>
                      {Math.abs(b.net) < 0.01 ? (
                        <span className="text-muted">settled up</span>
                      ) : b.net > 0 ? (
                        <span className="font-medium text-primary">is owed {formatINR(b.net)}</span>
                      ) : (
                        <span className="font-medium text-danger">owes {formatINR(-b.net)}</span>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </section>

            <section className="mt-4 rounded-lg border border-border bg-surface p-4">
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-base font-semibold">Expenses</h2>
                <Button size="sm" onClick={() => setShowExpenseForm((v) => !v)} disabled={!members || members.length === 0}>
                  <Plus className="h-4 w-4" />
                  Add expense
                </Button>
              </div>

              {showExpenseForm && members ? (
                <AddExpenseForm
                  groupID={id}
                  members={members}
                  categories={categories}
                  currentUserID={user?.id ?? members[0]?.user_id ?? ""}
                  onCancel={() => setShowExpenseForm(false)}
                  onCreated={async () => {
                    setShowExpenseForm(false);
                    await loadExpenses();
                  }}
                />
              ) : null}

              <div className="mt-4">
                {expenses === null ? (
                  <p className="text-sm text-muted">Loading…</p>
                ) : expenses.length === 0 ? (
                  <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed border-border px-4 py-8 text-center">
                    <Receipt aria-hidden className="h-6 w-6 text-muted" />
                    <p className="text-sm text-muted">No expenses yet. Add the first one for this group.</p>
                  </div>
                ) : (
                  <div className="grid gap-3">
                    {expenses.map((expense) => (
                      <div
                        key={expense.id}
                        className="flex items-center justify-between gap-3 border-b border-border pb-3 last:border-0 last:pb-0"
                      >
                        <div className="min-w-0">
                          <p className="truncate font-medium">{expense.title}</p>
                          <p className="text-xs text-muted">
                            Paid by {memberName(expense.paid_by_user_id)} · {expense.expense_date}
                            {categoryName(expense.category_slug) ? ` · ${categoryName(expense.category_slug)}` : ""}
                          </p>
                        </div>
                        <div className="flex shrink-0 items-center gap-3">
                          <span className="font-semibold">{formatINR(expense.amount)}</span>
                          <Button
                            variant="ghost"
                            size="icon"
                            aria-label="Delete expense"
                            title="Delete expense"
                            onClick={() => handleDeleteExpense(expense.id)}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </section>
          </>
        )}
      </div>
    </main>
  );
}

function AddExpenseForm({
  groupID,
  members,
  categories,
  currentUserID,
  onCancel,
  onCreated
}: {
  groupID: string;
  members: Member[];
  categories: Category[];
  currentUserID: string;
  onCancel: () => void;
  onCreated: () => void;
}) {
  const [title, setTitle] = useState("");
  const [amount, setAmount] = useState("");
  const [categorySlug, setCategorySlug] = useState("");
  const [paidBy, setPaidBy] = useState(currentUserID);
  const [splitMethod, setSplitMethod] = useState<SplitMethod>("equal");
  const [participants, setParticipants] = useState<Record<string, ParticipantState>>(() =>
    Object.fromEntries(members.map((m) => [m.user_id, { selected: true, amount: "", percentage: "", shares: "1" }]))
  );
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  function toggleParticipant(userID: string) {
    setParticipants((prev) => ({ ...prev, [userID]: { ...prev[userID], selected: !prev[userID].selected } }));
  }

  function updateParticipantField(userID: string, field: "amount" | "percentage" | "shares", value: string) {
    setParticipants((prev) => ({ ...prev, [userID]: { ...prev[userID], [field]: value } }));
  }

  function buildParticipantsPayload(totalAmount: number) {
    const selected = members.filter((m) => participants[m.user_id]?.selected);
    if (selected.length === 0) {
      setFormError("Select at least one participant.");
      return null;
    }

    if (splitMethod === "equal") {
      return selected.map((m) => ({ user_id: m.user_id }));
    }

    if (splitMethod === "unequal") {
      const amounts = selected.map((m) => Number.parseFloat(participants[m.user_id].amount || "0"));
      if (amounts.some((a) => !(a > 0))) {
        setFormError("Enter an amount greater than zero for every selected participant.");
        return null;
      }
      const sum = amounts.reduce((a, b) => a + b, 0);
      if (Math.abs(sum - totalAmount) > 0.01) {
        setFormError(`The amounts must add up to ${formatINR(totalAmount)} (currently ${formatINR(sum)}).`);
        return null;
      }
      return selected.map((m, i) => ({ user_id: m.user_id, amount: amounts[i] }));
    }

    if (splitMethod === "percentage") {
      const percentages = selected.map((m) => Number.parseFloat(participants[m.user_id].percentage || "0"));
      if (percentages.some((p) => !(p >= 0))) {
        setFormError("Enter a percentage for every selected participant.");
        return null;
      }
      const sum = percentages.reduce((a, b) => a + b, 0);
      if (Math.abs(sum - 100) > 0.01) {
        setFormError(`The percentages must add up to 100 (currently ${sum}).`);
        return null;
      }
      return selected.map((m, i) => ({ user_id: m.user_id, percentage: percentages[i] }));
    }

    const shares = selected.map((m) => Number.parseInt(participants[m.user_id].shares || "0", 10));
    if (shares.some((s) => !(s > 0))) {
      setFormError("Enter a share count of at least 1 for every selected participant.");
      return null;
    }
    return selected.map((m, i) => ({ user_id: m.user_id, shares: shares[i] }));
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");

    const totalAmount = Number.parseFloat(amount);
    if (!title.trim()) {
      setFormError("Enter a title.");
      return;
    }
    if (!(totalAmount > 0)) {
      setFormError("Enter an amount greater than zero.");
      return;
    }

    const participantsPayload = buildParticipantsPayload(totalAmount);
    if (!participantsPayload) return;

    setSubmitting(true);
    try {
      await apiPostAuthed(`/groups/${groupID}/expenses`, {
        title: title.trim(),
        amount: totalAmount,
        category_slug: categorySlug || undefined,
        paid_by_user_id: paidBy,
        split_method: splitMethod,
        expense_date: new Date().toISOString().slice(0, 10),
        participants: participantsPayload
      });
      onCreated();
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Unable to add expense.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="mt-4 grid gap-4 rounded-lg border border-border bg-background p-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <Input label="Title" name="title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Dinner at the beach shack" required />
        <Input
          label="Amount"
          name="amount"
          type="number"
          min="0.01"
          step="0.01"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="1200"
          required
        />
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <label className="grid gap-2 text-sm font-medium text-foreground">
          <span>Category</span>
          <select
            value={categorySlug}
            onChange={(e) => setCategorySlug(e.target.value)}
            className="h-11 rounded-lg border border-border bg-surface px-3 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
          >
            <option value="">No category</option>
            {categories.map((cat) => (
              <option key={cat.slug} value={cat.slug}>
                {cat.name}
              </option>
            ))}
          </select>
        </label>

        <label className="grid gap-2 text-sm font-medium text-foreground">
          <span>Paid by</span>
          <select
            value={paidBy}
            onChange={(e) => setPaidBy(e.target.value)}
            className="h-11 rounded-lg border border-border bg-surface px-3 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
          >
            {members.map((m) => (
              <option key={m.user_id} value={m.user_id}>
                {m.full_name}
              </option>
            ))}
          </select>
        </label>
      </div>

      <label className="grid gap-2 text-sm font-medium text-foreground">
        <span>Split</span>
        <select
          value={splitMethod}
          onChange={(e) => setSplitMethod(e.target.value as SplitMethod)}
          className="h-11 rounded-lg border border-border bg-surface px-3 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
        >
          <option value="equal">Equally</option>
          <option value="unequal">By exact amounts</option>
          <option value="percentage">By percentage</option>
          <option value="shares">By shares</option>
        </select>
      </label>

      <div className="grid gap-2">
        <span className="text-sm font-medium text-foreground">Participants</span>
        <div className="grid gap-2 rounded-lg border border-border bg-surface p-3">
          {members.map((m) => {
            const p = participants[m.user_id];
            return (
              <div key={m.user_id} className="flex items-center gap-3">
                <label className="flex flex-1 items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={p.selected}
                    onChange={() => toggleParticipant(m.user_id)}
                    className="h-4 w-4 rounded border-border"
                  />
                  {m.full_name}
                </label>
                {p.selected && splitMethod === "unequal" ? (
                  <input
                    type="number"
                    min="0.01"
                    step="0.01"
                    value={p.amount}
                    onChange={(e) => updateParticipantField(m.user_id, "amount", e.target.value)}
                    placeholder="Amount"
                    className="h-9 w-28 rounded-lg border border-border bg-background px-2 text-sm text-foreground outline-none focus:border-primary"
                  />
                ) : null}
                {p.selected && splitMethod === "percentage" ? (
                  <input
                    type="number"
                    min="0"
                    step="0.01"
                    value={p.percentage}
                    onChange={(e) => updateParticipantField(m.user_id, "percentage", e.target.value)}
                    placeholder="%"
                    className="h-9 w-20 rounded-lg border border-border bg-background px-2 text-sm text-foreground outline-none focus:border-primary"
                  />
                ) : null}
                {p.selected && splitMethod === "shares" ? (
                  <input
                    type="number"
                    min="1"
                    step="1"
                    value={p.shares}
                    onChange={(e) => updateParticipantField(m.user_id, "shares", e.target.value)}
                    placeholder="Shares"
                    className="h-9 w-20 rounded-lg border border-border bg-background px-2 text-sm text-foreground outline-none focus:border-primary"
                  />
                ) : null}
              </div>
            );
          })}
        </div>
      </div>

      {formError ? <p className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">{formError}</p> : null}

      <div className="flex items-center gap-2">
        <Button type="submit" disabled={submitting}>
          {submitting ? "Adding…" : "Add expense"}
        </Button>
        <Button type="button" variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
}

export default function GroupDetailPage() {
  return (
    <RequireAuth>
      <GroupDetail />
    </RequireAuth>
  );
}
