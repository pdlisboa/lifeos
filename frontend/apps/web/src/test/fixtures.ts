import type { components } from "@lifeos/api-client";

type User = components["schemas"]["User"];
type GoalSummary = components["schemas"]["GoalSummary"];
type Goal = components["schemas"]["Goal"];
type Competency = components["schemas"]["Competency"];
type NextAction = components["schemas"]["NextAction"];
type TodayGoal = components["schemas"]["TodayGoal"];
type DeltaPanel = components["schemas"]["DeltaPanel"];
type Track = components["schemas"]["Track"];
type Milestone = components["schemas"]["Milestone"];
type Probe = components["schemas"]["Probe"];
type ProbeTurn = components["schemas"]["ProbeTurn"];
type Consistency = components["schemas"]["Consistency"];
type Evidence = components["schemas"]["Evidence"];
type Session = components["schemas"]["Session"];

let seq = 0;
/** IDs previsíveis e distintos entre fixtures, sem precisar de uuid de verdade nos testes. */
function nextId(prefix: string): string {
  seq += 1;
  return `${prefix}-${seq}`;
}

export function makeUser(overrides: Partial<User> = {}): User {
  return {
    id: nextId("user"),
    email: "dev@lifeos.local",
    timezone: "America/Sao_Paulo",
    quietHoursFrom: null,
    quietHoursTo: null,
    ...overrides,
  };
}

export function makeGoalSummary(overrides: Partial<GoalSummary> = {}): GoalSummary {
  return {
    id: nextId("goal"),
    title: "Go backend",
    archetype: "skill",
    packId: "golang",
    status: "active",
    scopeMode: "normal",
    daysActive: 5,
    lastActivityAt: "2026-08-09T12:00:00Z",
    ...overrides,
  };
}

export function makeCompetency(overrides: Partial<Competency> = {}): Competency {
  return {
    id: nextId("comp"),
    packKey: "concurrency",
    label: "Concorrência",
    weight: 0.25,
    level: null,
    levelDescriptor: null,
    baselineLevel: null,
    confidence: "unknown",
    lastEvidenceAt: null,
    staleDays: null,
    retired: false,
    ...overrides,
  };
}

export function makeGoal(overrides: Partial<Goal> = {}): Goal {
  const summary = makeGoalSummary(overrides);
  return {
    ...summary,
    why: null,
    definitionOfDone: null,
    horizonOn: null,
    pausedUntil: null,
    competencies: [makeCompetency()],
    readyToActivate: false,
    blockers: ["falta a definição de pronto"],
    ...overrides,
  };
}

export function makeAction(overrides: Partial<NextAction> = {}): NextAction {
  return {
    id: nextId("action"),
    goalId: nextId("goal"),
    title: "Escreva uma função que lê 3 URLs em paralelo.",
    detail: "Um problema fechado, resolvível em um arquivo.",
    practiceFormat: "kata",
    estimatedMin: 30,
    minimalVariant: "Versão de 5 min: releia e escreva só o primeiro passo.",
    competencyId: nextId("comp"),
    milestoneId: nextId("milestone"),
    generatedBy: "fallback",
    originKind: "practice",
    status: "pending",
    createdAt: "2026-08-09T12:00:00Z",
    ...overrides,
  };
}

export function makeTodayGoal(overrides: Partial<TodayGoal> = {}): TodayGoal {
  const goal = overrides.goal ?? makeGoalSummary();
  return {
    goal,
    action: makeAction({ goalId: goal.id }),
    recentWin: null,
    ...overrides,
  };
}

export function makeConsistency(overrides: Partial<Consistency> = {}): Consistency {
  return {
    activeDays: 1,
    windowDays: 30,
    todayDone: true,
    label: "1 dos últimos 30 dias",
    ...overrides,
  };
}

export function makeMilestone(overrides: Partial<Milestone> = {}): Milestone {
  return {
    id: nextId("milestone"),
    ordinal: 1,
    title: "CLI que lê arquivo e agrega dados, com testes",
    completionCriteria: "Roda com `go run` e tem testes table-driven.",
    status: "current",
    completedAt: null,
    competencyIds: [],
    ...overrides,
  };
}

export function makeTrack(overrides: Partial<Track> = {}): Track {
  return {
    id: nextId("track"),
    version: 1,
    milestones: [makeMilestone()],
    ...overrides,
  };
}

export function makeDeltaCompetency(
  overrides: Partial<DeltaPanel["competencies"][number]> = {},
): DeltaPanel["competencies"][number] {
  return {
    ...makeCompetency(),
    delta: null,
    deltaWindowDays: null,
    risesLast90d: 0,
    ...overrides,
  };
}

export function makeDeltaPanel(overrides: Partial<DeltaPanel> = {}): DeltaPanel {
  return {
    goalId: nextId("goal"),
    daysActive: 5,
    competencies: [makeDeltaCompetency()],
    totals: { evidenceCount: 0, milestonesDone: 0, sessionMinutes: 0 },
    headline: null,
    ...overrides,
  };
}

export function makeProbeTurn(overrides: Partial<ProbeTurn> = {}): ProbeTurn {
  return {
    id: nextId("turn"),
    ordinal: 1,
    question: "Já mexeu com goroutines?",
    answer: null,
    competencyId: nextId("comp"),
    inferredLevel: null,
    answeredAt: null,
    ...overrides,
  };
}

export function makeProbe(overrides: Partial<Probe> = {}): Probe {
  const turn = makeProbeTurn();
  return {
    id: nextId("probe"),
    status: "open",
    askedCount: 1,
    maxQuestions: 5,
    currentQuestion: turn,
    turns: [turn],
    canSkip: true,
    ...overrides,
  };
}

export function makeEvidence(overrides: Partial<Evidence> = {}): Evidence {
  return {
    id: nextId("evidence"),
    kind: "written_text",
    title: "primeira evidência",
    body: "texto de evidência",
    blobUrl: null,
    externalUrl: null,
    language: null,
    supersedesId: null,
    createdAt: "2026-08-09T12:00:00Z",
    ...overrides,
  };
}

export function makeSession(overrides: Partial<Session> = {}): Session {
  return {
    id: nextId("session"),
    startedAt: "2026-08-09T12:00:00Z",
    durationMin: 25,
    localOn: "2026-08-09",
    note: null,
    energy: null,
    ...overrides,
  };
}
