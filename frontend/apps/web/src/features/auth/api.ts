import { api, unwrap } from "@/lib/api";
import type { components } from "@lifeos/api-client";

export type User = components["schemas"]["User"];

export function fetchMe() {
  return unwrap(api.GET("/me"));
}

export function login(email: string, password: string) {
  return unwrap(
    api.POST("/auth/login", {
      body: { email, password, client: "web" },
    }),
  );
}

export function logout() {
  return api.POST("/auth/logout");
}
