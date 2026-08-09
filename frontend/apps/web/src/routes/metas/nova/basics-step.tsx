import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button, Input, Label } from "@/components/ui";
import { ProblemError } from "@/components/problem-error";
import { useCreateGoal } from "@/features/goal/use-goals";
import { packs } from "./packs";

const schema = z.object({
  title: z.string().min(3, "pelo menos 3 caracteres"),
});

type FormValues = z.infer<typeof schema>;

export function BasicsStep({ onCreated }: { onCreated: (goalId: string) => void }) {
  const [packId, setPackId] = useState(packs[0].packId);
  const createGoal = useCreateGoal();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({ resolver: zodResolver(schema) });

  const onSubmit = handleSubmit(async ({ title }) => {
    try {
      const pack = packs.find((p) => p.packId === packId)!;
      const result = await createGoal.mutateAsync({ title, archetype: pack.archetype, packId: pack.packId });
      if (!result.goal) throw new Error("resposta sem goal");
      onCreated(result.goal.id);
    } catch {
      // erro fica em createGoal.error, renderizado abaixo
    }
  });

  return (
    <form onSubmit={onSubmit} className="space-y-6">
      <div>
        <Label htmlFor="title">O que você quer aprender?</Label>
        <Input id="title" placeholder="Go" {...register("title")} />
        {errors.title && <p className="mt-1 text-sm text-danger-fg">{errors.title.message}</p>}
      </div>

      <div>
        <Label>Área</Label>
        <div className="flex gap-4">
          {packs.map((pack) => (
            <label key={pack.packId} className="flex items-center gap-2 text-sm text-fg-primary">
              <input
                type="radio"
                name="packId"
                value={pack.packId}
                checked={packId === pack.packId}
                onChange={() => setPackId(pack.packId)}
              />
              {pack.label}
            </label>
          ))}
        </div>
      </div>

      {createGoal.error && <ProblemError error={createGoal.error} />}

      <div className="flex justify-end">
        <Button type="submit" disabled={createGoal.isPending}>
          {createGoal.isPending ? "Criando…" : "Continuar"}
        </Button>
      </div>
    </form>
  );
}
