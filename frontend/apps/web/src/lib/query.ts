import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "@lifeos/api-client";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        // 401/404/409/422 são estado, não falha transitória — não faz sentido retentar.
        if (error instanceof ApiError && error.problem.status < 500) return false;
        return failureCount < 2;
      },
      staleTime: 10_000,
    },
  },
});
