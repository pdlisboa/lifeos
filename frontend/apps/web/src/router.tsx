import { createBrowserRouter } from "react-router-dom";
import { RootLayout } from "@/routes/root-layout";
import { RequireAuth } from "@/components/require-auth";
import { LoginRoute } from "@/routes/login";
import { HojeRoute } from "@/routes/hoje";
import { MetasListRoute } from "@/routes/metas";
import { NovaMetaRoute } from "@/routes/metas/nova";
import { MetaDetailRoute } from "@/routes/metas/goal";
import { EvidenciaRoute } from "@/routes/metas/goal/evidencia";

export const router = createBrowserRouter([
  { path: "/login", element: <LoginRoute /> },
  {
    element: <RequireAuth />,
    children: [
      {
        element: <RootLayout />,
        children: [
          { path: "/", element: <HojeRoute /> },
          { path: "/metas", element: <MetasListRoute /> },
          { path: "/metas/nova", element: <NovaMetaRoute /> },
          { path: "/metas/:goalId", element: <MetaDetailRoute /> },
          { path: "/metas/:goalId/evidencia", element: <EvidenciaRoute /> },
        ],
      },
    ],
  },
]);
