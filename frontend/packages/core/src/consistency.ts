/**
 * "18 dos últimos 30 dias" — nunca uma streak (RN-11, P6). Não existe versão
 * desta função que zere; se aparecer um `<StreakCounter>` em algum componente,
 * é sinal de que esta função foi contornada.
 */
export function formatConsistency(activeDays: number, windowDays: number): string {
  return `${activeDays} dos últimos ${windowDays} dias`;
}
