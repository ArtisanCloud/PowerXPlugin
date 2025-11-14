import { useAuth } from "~/composables/useAuth";

export default defineNuxtRouteMiddleware(async (to) => {
  if (to.path.startsWith("/users")) {
    return;
  }
  if (!process.client) return;

  const auth = useAuth();
  await auth.ensureFreshToken();

  if (!auth.token.value) {
    return navigateTo({
      path: "/users/login",
      query: { redirect: to.fullPath },
    });
  }
});
