import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { Button, Input, Label, Textarea } from "@/components/ui";
import { ProblemError } from "@/components/problem-error";
import { useGoal } from "@/features/goal/use-goals";
import { useGoalAction } from "@/features/action/use-actions";
import { useCreateEvidence } from "@/features/evidence/use-evidence";
import { cn } from "@/lib/cn";
import type { EvidenceKind } from "@/features/evidence/api";

const kinds: { value: EvidenceKind; label: string }[] = [
  { value: "code_snippet", label: "Código" },
  { value: "written_text", label: "Texto" },
  { value: "repo_link", label: "Link" },
];

export function EvidenciaRoute() {
  const { goalId } = useParams<{ goalId: string }>();
  const navigate = useNavigate();
  const { data: goal } = useGoal(goalId!);
  const { data: action } = useGoalAction(goalId!);

  const [kind, setKind] = useState<EvidenceKind>("code_snippet");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [externalUrl, setExternalUrl] = useState("");
  const [linkToAction, setLinkToAction] = useState(true);

  const createEvidence = useCreateEvidence(goalId!);

  const submit = async () => {
    await createEvidence.mutateAsync({
      kind,
      title: title.trim() || undefined,
      body: kind !== "repo_link" ? body.trim() || undefined : undefined,
      externalUrl: kind === "repo_link" ? externalUrl.trim() || undefined : undefined,
      actionId: linkToAction ? action?.id : undefined,
    });
    navigate(`/metas/${goalId}`);
  };

  return (
    <div className="mx-auto max-w-lg space-y-6">
      <Link to={`/metas/${goalId}`} className="text-sm text-fg-muted hover:text-fg-secondary">
        ← {goal?.title ?? "Meta"}
      </Link>

      <h1 className="text-lg font-medium text-fg-primary">Nova evidência{goal ? ` · ${goal.title}` : ""}</h1>

      <div className="flex gap-2">
        {kinds.map((k) => (
          <button
            key={k.value}
            className={cn(
              "rounded-md border px-3 py-1.5 text-sm",
              kind === k.value
                ? "border-delta-positive bg-delta-positive-muted text-fg-primary"
                : "border-border text-fg-secondary hover:text-fg-primary",
            )}
            onClick={() => setKind(k.value)}
          >
            {k.label}
          </button>
        ))}
      </div>

      <div>
        <Label htmlFor="title">Título (opcional)</Label>
        <Input id="title" value={title} onChange={(e) => setTitle(e.target.value)} />
      </div>

      {kind === "repo_link" ? (
        <div>
          <Label htmlFor="externalUrl">Link</Label>
          <Input
            id="externalUrl"
            type="url"
            value={externalUrl}
            onChange={(e) => setExternalUrl(e.target.value)}
            placeholder="https://github.com/..."
          />
        </div>
      ) : (
        <div>
          <Label htmlFor="body">{kind === "code_snippet" ? "Código" : "Texto"}</Label>
          <Textarea
            id="body"
            rows={10}
            value={body}
            onChange={(e) => setBody(e.target.value)}
            className="font-mono"
          />
        </div>
      )}

      {action && (
        <label className="flex items-center gap-2 text-sm text-fg-secondary">
          <input type="checkbox" checked={linkToAction} onChange={(e) => setLinkToAction(e.target.checked)} />
          Ligada à ação de hoje: {action.title}
        </label>
      )}

      {createEvidence.error && <ProblemError error={createEvidence.error} />}

      <div className="flex justify-end">
        <Button disabled={createEvidence.isPending} onClick={submit}>
          {createEvidence.isPending ? "Registrando…" : "Registrar"}
        </Button>
      </div>
    </div>
  );
}
