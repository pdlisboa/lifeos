import { useState } from "react";
import { BasicsStep } from "./basics-step";
import { ProbeStep } from "./probe-step";
import { DoneStep } from "./done-step";

type Step = "basics" | "probe" | "done";

export function NovaMetaRoute() {
  const [step, setStep] = useState<Step>("basics");
  const [goalId, setGoalId] = useState<string | null>(null);

  return (
    <div className="mx-auto max-w-md">
      {step === "basics" && (
        <BasicsStep
          onCreated={(id) => {
            setGoalId(id);
            setStep("probe");
          }}
        />
      )}
      {step === "probe" && goalId && <ProbeStep goalId={goalId} onDone={() => setStep("done")} />}
      {step === "done" && goalId && <DoneStep goalId={goalId} />}
    </div>
  );
}
