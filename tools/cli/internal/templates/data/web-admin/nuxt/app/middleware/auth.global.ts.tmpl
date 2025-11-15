import { useAuth } from "~/composables/useAuth";

const PUBLIC_ROUTE_PREFIXES = ["/users"];

export default defineNuxtRouteMiddleware(async (to) => {
  if (!process.client) return;

  if (to.meta?.public === true) {
    return;
  }

  if (PUBLIC_ROUTE_PREFIXES.some((prefix) => to.path.startsWith(prefix))) {
    return;
  }

  const auth = useAuth();
  await auth.ensureFreshToken();

  if (!auth.token.value) {
    return navigateTo({
      path: "/users/login",
      query: { redirect: to.fullPath },
    });
  }
});
