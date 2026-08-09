/** Formato mínimo aceito por formatProjectionFooter — qualquer objeto com esses campos serve. */
export interface ProjectionForFooter {
  available: boolean;
  reason: string | null;
  minutesPerWeek: number | null;
}

/**
 * O rodapé do Painel de Delta (§5.1/§7.2): ritmo real, nunca uma data de
 * chegada inventada. Sem estimativa de esforço restante por marco no
 * modelo, o texto não promete "sai em N semanas" — diz o ritmo e para aí.
 * Quando `available` é false, `reason` já vem pronto do servidor (ex.:
 * "ainda coletando ritmo (2 de 3 semanas)") — usa direto, sem reformatar.
 */
export function formatProjectionFooter(p: ProjectionForFooter): string {
  if (!p.available || p.minutesPerWeek === null) {
    return p.reason ?? "ainda coletando ritmo";
  }
  const hoursPerWeek = (p.minutesPerWeek / 60).toFixed(1).replace(".", ",");
  return `No seu ritmo (~${hoursPerWeek}h/semana). Ainda não há estimativa de quando chega o próximo marco.`;
}
