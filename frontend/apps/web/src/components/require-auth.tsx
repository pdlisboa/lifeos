import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useCurrentUser } from "@/features/auth/use-auth";

/**
 * Sessão expirada redireciona pro login preservando a rota que você tentou
 * abrir (06-frontend.md §10, ponto em aberto #2) — LoginRoute usa `state.from`
 * pra voltar pra lá depois de autenticar.
 */
export function RequireAuth() {
  const location = useLocation();
  const { data: user, isPending, isError } = useCurrentUser();

  if (isPending) return null;
  if (isError || !user) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }
  return <Outlet />;
}
