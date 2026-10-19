import { ref, onScopeDispose } from 'vue';
import type { RegressionClient, CaseDefinition, ProfileDefinition, Run, RunDetail, CreateRun, SuiteDefinition } from '@artisan-cloud/plugin-framework-client';

export function usePowerXRegression(client: RegressionClient) {
  const suites = ref<SuiteDefinition[]>([]);
  const cases = ref<CaseDefinition[]>([]), profiles = ref<ProfileDefinition[]>([]), runs = ref<Run[]>([]);
  const detail = ref<RunDetail | null>(null), error = ref(''), loading = ref(false);
  const connection = ref<'idle' | 'connected' | 'error'>('idle');
  let stream: AbortController | null = null, generation = 0;
  const fail = (e: any, operation: 'catalog' | 'request' = 'request') => {
    const code = e?.data?.error?.code || e?.message;
    const status = e?.response?.status ?? e?.statusCode ?? e?.status;
    if (/^REGRESSION_[A-Z_]+$/.test(code || '')) error.value = code;
    else if (operation === 'catalog' && status === 404) error.value = 'REGRESSION_API_UNAVAILABLE';
    else error.value = 'REGRESSION_REQUEST_FAILED';
  };
  function stop() { stream?.abort(); stream = null; connection.value = 'idle'; }
  function reset() { generation++; stop(); suites.value = []; cases.value = []; profiles.value = []; runs.value = []; detail.value = null; error.value = ''; loading.value = false; }
  async function load() {
    const revision = generation; loading.value = true; error.value = '';
    try {
      const [c, p, r, s] = await Promise.all([client.cases(), client.profiles(), client.runs(), client.suites()]);
      if (revision === generation) { suites.value = s.items; cases.value = c.items; profiles.value = p.items; runs.value = r.items; }
    } catch (e) { if (revision === generation) fail(e, 'catalog'); }
    finally { if (revision === generation) loading.value = false; }
  }
  async function watch(snapshot: RunDetail) {
    stop(); detail.value = snapshot;
    const controller = new AbortController(); stream = controller; connection.value = 'connected';
    try {
      await client.watch(snapshot, controller.signal, next => { if (stream === controller) { detail.value = next; runs.value = runs.value.map(r => r.uuid === next.run.uuid ? next.run : r); } });
      if (stream === controller) connection.value = 'idle';
    } catch (e) { if (stream === controller && !controller.signal.aborted) { connection.value = 'error'; fail(e); } }
  }
  async function open(id: string) {
    const revision = generation; error.value = '';
    try { const d = await client.get(id); if (revision === generation) void watch(d); }
    catch (e) { if (revision === generation) fail(e); }
  }
  async function run(body: CreateRun) {
    const revision = generation; error.value = ''; loading.value = true;
    try { const d = await client.create(body); if (revision === generation) { void watch(d); runs.value = [d.run, ...runs.value.filter(r => r.uuid !== d.run.uuid)]; } }
    catch (e) { if (revision === generation) fail(e); }
    finally { if (revision === generation) loading.value = false; }
  }
  async function cancel() { if (!detail.value) return; const revision = generation; try { const d = await client.cancel(detail.value.run.uuid); if (revision === generation && (!detail.value || d.run.revision >= detail.value.run.revision)) detail.value = d; } catch (e) { if (revision === generation) fail(e); } }
  async function cleanup() { if (!detail.value) return; const revision = generation; try { const d = await client.cleanup(detail.value.run.uuid); if (revision === generation) void watch(d); } catch (e) { if (revision === generation) fail(e); } }
  async function reconnect() { if (detail.value) await open(detail.value.run.uuid); }
  async function download() {
    if (!detail.value) return; const revision = generation;
    try { const report = await client.report(detail.value.run.uuid); if (revision !== generation) return; const url = URL.createObjectURL(new Blob([JSON.stringify(report, null, 2)], { type: 'application/json' })); const a = document.createElement('a'); a.href = url; a.download = 'regression-report.json'; a.click(); URL.revokeObjectURL(url); }
    catch (e) { if (revision === generation) fail(e); }
  }
  onScopeDispose(reset);
  return { suites, cases, profiles, runs, detail, error, loading, connection, load, open, run, cancel, cleanup, reconnect, download, stop, reset };
}
