import { useState } from "react";
import { Button, Input, Label } from "@/components/ui";
import { ProblemError } from "@/components/problem-error";
import { useCreateSession } from "@/features/session/use-session";

export function SessionForm({ goalId, onDone }: { goalId: string; onDone: () => void }) {
  const [durationMin, setDurationMin] = useState(25);
  const [note, setNote] = useState("");
  const createSession = useCreateSession(goalId);

  const submit = async () => {
    try {
      await createSession.mutateAsync({
        startedAt: new Date().toISOString(),
        durationMin,
        note: note.trim() || undefined,
      });
      onDone();
    } catch {
      // erro fica em createSession.error, renderizado abaixo
    }
  };

  return (
    <div className="space-y-3 rounded-md border border-border-subtle bg-bg-overlay p-4">
      <div>
        <Label htmlFor="durationMin">Quantos minutos?</Label>
        <Input
          id="durationMin"
          type="number"
          min={1}
          max={600}
          value={durationMin}
          onChange={(e) => setDurationMin(Number(e.target.value))}
        />
      </div>
      <div>
        <Label htmlFor="note">Nota (opcional)</Label>
        <Input id="note" value={note} onChange={(e) => setNote(e.target.value)} />
      </div>
      {createSession.error && <ProblemError error={createSession.error} />}
      <div className="flex gap-2">
        <Button disabled={createSession.isPending} onClick={submit}>
          {createSession.isPending ? "Registrando…" : "Registrar sessão"}
        </Button>
        <Button variant="ghost" onClick={onDone}>
          cancelar
        </Button>
      </div>
    </div>
  );
}
