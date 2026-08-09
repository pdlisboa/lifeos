import { api, unwrap } from "@/lib/api";

export function fetchToday() {
  return unwrap(api.GET("/today"));
}
