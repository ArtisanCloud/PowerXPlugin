import { useAuth } from "~/composables/useAuth";
import { useUserStore } from "~/stores/user";

const PUBLIC_ROUTE_PREFIXES = ["/users"];
const ROOT_ONLY_ROUTE_PREFIXES = [
  "/templates",
  "/capabilities",
  "/powerx/capability-lab",
];

export default defineNuxtRouteMiddleware(async (to) => {
  if (!process.client) return;
  const runtimeConfig = useRuntimeConfig();
  const insidePowerX =
    runtimeConfig.public?.insidePowerX === true ||
    runtimeConfig.public?.insidePowerX === "true";

  if (to.meta?.public === true) {
    return;
  }

  if (PUBLIC_ROUTE_PREFIXES.some((prefix) => to.path.startsWith(prefix))) {
    return;
  }

  const auth = useAuth();
  await auth.ensureFreshToken();

  if (!auth.token.value) {
    if (insidePowerX) {
      if (!auth.delegatedError?.value) {
        auth.rememberAuthError?.("PowerX 会话已失效，请回到宿主重新登录");
      }
      return;
    }
    return navigateTo({
      path: "/users/login",
      query: { redirect: to.fullPath },
    });
  }

  const userStore = useUserStore();
  if (!userStore.context && !userStore.isLoading) {
    try {
      await userStore.fetchUserContext();
    } catch {
      return navigateTo({
        path: "/users/login",
        query: { redirect: to.fullPath },
      });
    }
  }
  if (
    ROOT_ONLY_ROUTE_PREFIXES.some((prefix) => to.path.startsWith(prefix)) &&
    !userStore.isRoot
  ) {
    return navigateTo("/intro");
  }
});
