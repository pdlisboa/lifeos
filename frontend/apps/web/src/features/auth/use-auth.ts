import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as authApi from "./api";

export const meQueryKey = ["me"] as const;

/** `retry: false` de propósito — 401 aqui é "não logado", não uma falha transitória. */
export function useCurrentUser() {
  return useQuery({
    queryKey: meQueryKey,
    queryFn: authApi.fetchMe,
    retry: false,
  });
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ email, password }: { email: string; password: string }) => authApi.login(email, password),
    onSuccess: (data) => {
      qc.setQueryData(meQueryKey, data.user);
    },
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: authApi.logout,
    onSuccess: () => {
      qc.clear();
    },
  });
}
