import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { Button, Input, Label } from "@/components/ui";
import { useCurrentUser, useLogin } from "@/features/auth/use-auth";
import { ApiError } from "@lifeos/api-client";

const schema = z.object({
  email: z.string().email("email inválido"),
  password: z.string().min(1, "obrigatório"),
});

type FormValues = z.infer<typeof schema>;

export function LoginRoute() {
  const navigate = useNavigate();
  const location = useLocation();
  const { data: user, isPending: checkingSession } = useCurrentUser();
  const login = useLogin();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({ resolver: zodResolver(schema) });

  const from = (location.state as { from?: { pathname: string } } | null)?.from?.pathname ?? "/";

  // Já logado (ex.: voltou pra /login por engano) — não mostra o form de novo.
  if (!checkingSession && user) {
    return <Navigate to={from} replace />;
  }

  const onSubmit = handleSubmit(async (values) => {
    try {
      await login.mutateAsync(values);
      navigate(from, { replace: true });
    } catch {
      // erro fica em login.error, renderizado abaixo
    }
  });

  return (
    <div className="flex min-h-screen items-center justify-center bg-bg-base px-4">
      <div className="w-full max-w-sm">
        <h1 className="mb-6 text-center text-lg font-medium text-fg-primary">LifeOS</h1>
        <form onSubmit={onSubmit} className="space-y-4">
          <div>
            <Label htmlFor="email">Email</Label>
            <Input id="email" type="email" autoComplete="username" {...register("email")} />
            {errors.email && <p className="mt-1 text-sm text-danger-fg">{errors.email.message}</p>}
          </div>
          <div>
            <Label htmlFor="password">Senha</Label>
            <Input id="password" type="password" autoComplete="current-password" {...register("password")} />
            {errors.password && <p className="mt-1 text-sm text-danger-fg">{errors.password.message}</p>}
          </div>
          {login.error && (
            <p className="text-sm text-danger-fg">
              {login.error instanceof ApiError && login.error.problem.status === 401
                ? "Email ou senha incorretos."
                : "Não deu pra entrar agora. Tenta de novo."}
            </p>
          )}
          <Button type="submit" className="w-full" disabled={login.isPending}>
            {login.isPending ? "Entrando…" : "Entrar"}
          </Button>
        </form>
      </div>
    </div>
  );
}
