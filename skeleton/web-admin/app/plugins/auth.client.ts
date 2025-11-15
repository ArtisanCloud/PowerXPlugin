export default defineNuxtPlugin(() => {
  const auth = useAuth();
  if (process.client) {
    auth.initAuth();
  }
});
