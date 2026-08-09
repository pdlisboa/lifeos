import { NavLink, Outlet } from "react-router-dom";
import { cn } from "@/lib/cn";
import { useLogout } from "@/features/auth/use-auth";

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  cn(
    "rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
    isActive ? "bg-bg-overlay text-fg-primary" : "text-fg-secondary hover:text-fg-primary",
  );

export function RootLayout() {
  const logout = useLogout();

  return (
    <div className="min-h-screen bg-bg-base">
      <header className="border-b border-border-subtle">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-4 py-3">
          <nav className="flex items-center gap-1">
            <NavLink to="/" end className={navLinkClass}>
              Hoje
            </NavLink>
            <NavLink to="/metas" className={navLinkClass}>
              Metas
            </NavLink>
          </nav>
          <button
            onClick={() => logout.mutate()}
            className="text-sm text-fg-muted hover:text-fg-secondary"
          >
            Sair
          </button>
        </div>
      </header>
      <main className="mx-auto max-w-3xl px-4 py-8">
        <Outlet />
      </main>
    </div>
  );
}
