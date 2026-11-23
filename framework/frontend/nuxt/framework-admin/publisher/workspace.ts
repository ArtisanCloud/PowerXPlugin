import { computed, ref } from 'vue'
import {
  activeBuildStages,
  directBuildFeature,
  type PublisherArtifact,
  type PublisherBuild,
  type PublisherConnection,
  type PublisherPage,
  type PublisherPlan,
  type PublisherPlanInput
} from './types'
import { PublisherClientError, type PublisherClient } from './client'

export const publisherPolling = { interval: 2000, window: 30 * 60 * 1000 }
const emptyPage = <T>(): PublisherPage<T> => ({ items: [], total: 0, page: 1, size: 20 })
export function createPublisherWorkspace(client: PublisherClient) {
  const connection = ref<PublisherConnection | null>(null)
  const executor = ref<'native' | 'container'>('native')
  const environment = computed(() => connection.value?.info.build_environment)
  const executorUpdateRequired = computed(() =>
    !!connection.value && !connection.value.info.features.includes(directBuildFeature)
  )
  const needsReconcile = ref(false)
  const plans = ref(emptyPage<PublisherPlan>()),
    builds = ref(emptyPage<PublisherBuild>())
  const selectedPlan = ref<PublisherPlan | null>(null),
    selectedBuild = ref<PublisherBuild | null>(null)
  const ongoingBuild = ref<PublisherBuild | null>(null)
  const error = ref(''),
    busy = ref(''),
    pollPaused = ref(false),
    verified = ref<PublisherArtifact | null>(null)
  let timer: ReturnType<typeof setTimeout> | undefined,
    epoch = 0,
    disposed = false
  const isActive = (build: PublisherBuild | null) =>
    !!build && activeBuildStages.includes(build.stage)
  const running = computed(() => isActive(ongoingBuild.value))
  const supportedTarget = computed(() =>
    !!selectedPlan.value &&
    !!connection.value?.info.build_targets.some(
      (target) => target.os === selectedPlan.value?.target.os && target.arch === selectedPlan.value?.target.arch
    )
  )
  const canBuild = computed(() =>
    !executorUpdateRequired.value && supportedTarget.value &&
    (executor.value === 'container' ? !!environment.value?.container_ready : !!environment.value?.native_available)
  )
  const canPrepareContainer = computed(() =>
    !executorUpdateRequired.value && supportedTarget.value &&
    executor.value === 'container' &&
    environment.value?.container_available === true &&
    !environment.value.container_ready &&
    environment.value.can_start
  )
  function acceptConnection(value: PublisherConnection) {
    connection.value = value
  }
  function fail(cause: unknown) {
    error.value =
      cause instanceof PublisherClientError ? cause.code : 'PUBLISHER_PROXY_RESPONSE_INVALID'
    if (error.value === 'PUBLISHER_UNAUTHORIZED') connection.value = null
  }
  function stop() {
    epoch++
    clearTimeout(timer)
    timer = undefined
  }
  function accept(build: PublisherBuild) {
    if (
      selectedBuild.value?.build_id === build.build_id &&
      !isActive(selectedBuild.value) &&
      isActive(build)
    )
      return
    if (selectedBuild.value?.build_id === build.build_id) selectedBuild.value = build
    if (isActive(build)) ongoingBuild.value = build
    else if (ongoingBuild.value?.build_id === build.build_id) ongoingBuild.value = null
    const index = builds.value.items.findIndex((b) => b.build_id === build.build_id)
    if (index >= 0) builds.value.items[index] = build
  }
  function watchBuild() {
    stop()
    pollPaused.value = false
    if (!running.value || disposed) return
    const captured = epoch,
      id = ongoingBuild.value!.build_id,
      deadline = Date.now() + publisherPolling.window
    const tick = async () => {
      if (captured !== epoch || disposed) return
      if (Date.now() >= deadline) {
        pollPaused.value = true
        return
      }
      try {
        const build = await client.build(id)
        if (captured !== epoch || disposed) return
        accept(build)
        if (isActive(build)) timer = setTimeout(tick, publisherPolling.interval)
      } catch (cause) {
        if (captured === epoch && !disposed) {
          fail(cause)
          pollPaused.value = true
        }
      }
    }
    timer = setTimeout(tick, publisherPolling.interval)
  }
  async function act(name: string, action: () => Promise<void>) {
    if (busy.value || disposed) return false
    if (needsReconcile.value && ['create', 'run'].includes(name)) return false
    busy.value = name
    error.value = ''
    try {
      await action()
      return true
    } catch (cause) {
      fail(cause)
      if (
        ['create', 'run', 'cancel'].includes(name) &&
        ['PUBLISHER_PROXY_UNAVAILABLE', 'PUBLISHER_PROXY_RESPONSE_INVALID'].includes(error.value)
      )
        needsReconcile.value = true
      return false
    } finally {
      busy.value = ''
    }
  }
  async function refresh() {
    const [p, b] = await Promise.all([
      client.plans(plans.value.page),
      client.builds(builds.value.page)
    ])
    if (disposed) return
    plans.value = p
    builds.value = b
    if (ongoingBuild.value) accept(await client.build(ongoingBuild.value.build_id))
    else ongoingBuild.value = b.items.find((item) => isActive(item)) || null
    if (selectedPlan.value)
      selectedPlan.value = await client.plan(selectedPlan.value.release_plan_id)
    else selectedPlan.value = p.items[0] || null
    if (selectedBuild.value) accept(await client.build(selectedBuild.value.build_id))
    else
      selectedBuild.value =
        b.items.find((item) => item.release_plan_id === selectedPlan.value?.release_plan_id) || null
    watchBuild()
    needsReconcile.value = false
  }
  async function restoreReferences(planID?: string, buildID?: string) {
    if (planID) selectedPlan.value = await client.plan(planID)
    if (buildID) {
      selectedBuild.value = await client.build(buildID)
      if (isActive(selectedBuild.value)) ongoingBuild.value = selectedBuild.value
      selectedPlan.value = await client.plan(selectedBuild.value.release_plan_id)
    }
  }
  async function prepareContainer() {
    await client.startEnvironment()
    const deadline = Date.now() + 90000
    while (!disposed && Date.now() < deadline) {
      const value = await client.info()
      acceptConnection(value)
      if (value.info.build_environment.container_ready) return
      await new Promise(resolve => setTimeout(resolve, 1500))
    }
    if (!disposed) throw new PublisherClientError('PUBLISHER_CONTAINER_START_FAILED')
  }
  return {
    connection,
    executor,
    environment,
    executorUpdateRequired,
    needsReconcile,
    plans,
    builds,
    selectedPlan,
    selectedBuild,
    ongoingBuild,
    error,
    busy,
    pollPaused,
    verified,
    running,
    canBuild,
    canPrepareContainer,
    checkEnvironment: () => act('environment', async () => {
      acceptConnection(await client.info())
    }),
    reconnect: () => act('reconnect', async () => {
      try {
        acceptConnection(await client.reconnect())
      } catch (cause) {
        if (cause instanceof PublisherClientError && cause.code === 'PUBLISHER_NOT_FOUND')
          throw new PublisherClientError('PUBLISHER_EXECUTOR_HOST_UPDATE_REQUIRED')
        throw cause
      }
      await refresh()
    }),
    connect: (planID?: string, buildID?: string) =>
      act('connect', async () => {
        acceptConnection(await client.connect())
        await restoreReferences(planID, buildID)
        await refresh()
      }),
    restore: (planID?: string, buildID?: string) =>
      act('restore', async () => {
        if (!(await client.status()).connected) return
        try {
          acceptConnection(await client.info())
        } catch (cause) {
          if (cause instanceof PublisherClientError && cause.code === 'PUBLISHER_UNAUTHORIZED')
            return
          throw cause
        }
        await restoreReferences(planID, buildID)
        await refresh()
      }),
    refresh: () => act('refresh', refresh),
    create: (input: PublisherPlanInput) =>
      act('create', async () => {
        const p = await client.create(input)
        stop()
        plans.value.page = 1
        selectedPlan.value = p
        selectedBuild.value = null
        verified.value = null
        await refresh()
      }),
    selectPlan: (p: PublisherPlan) =>
      act('select', async () => {
        stop()
        selectedPlan.value = p
        verified.value = null
        selectedBuild.value =
          builds.value.items.find((b) => b.release_plan_id === p.release_plan_id) || null
        watchBuild()
      }),
    selectBuild: (b: PublisherBuild) =>
      act('select', async () => {
        stop()
        selectedPlan.value = await client.plan(b.release_plan_id)
        selectedBuild.value = b
        verified.value = null
        if (isActive(b)) ongoingBuild.value = b
        watchBuild()
      }),
    pagePlans: (page: number) =>
      act('page', async () => {
        plans.value = await client.plans(page)
      }),
    pageBuilds: (page: number) =>
      act('page', async () => {
        builds.value = await client.builds(page)
      }),
    run: async () => {
      if (busy.value || disposed || running.value || needsReconcile.value) return false
      const planID = selectedPlan.value?.release_plan_id
      if (canPrepareContainer.value) {
        if (!(await act('prepare-build', prepareContainer))) return false
        if (selectedPlan.value?.release_plan_id !== planID || executor.value !== 'container') return false
      }
      return act('run', async () => {
        if (executorUpdateRequired.value)
          throw new PublisherClientError('PUBLISHER_EXECUTOR_UPDATE_REQUIRED')
        if (!selectedPlan.value || !canBuild.value)
          throw new PublisherClientError('PUBLISHER_BUILD_PLATFORM_REQUIRED')
        const b = await client.run(selectedPlan.value.release_plan_id, executor.value)
        stop()
        selectedBuild.value = b
        ongoingBuild.value = isActive(b) ? b : null
        verified.value = null
        builds.value.page = 1
        builds.value = await client.builds()
        watchBuild()
      })
    },
    cancel: () =>
      act('cancel', async () => {
        const build = ongoingBuild.value
        if (!build) return
        stop()
        accept(await client.cancel(build.build_id))
      }),
    verify: () =>
      act('verify', async () => {
        if (!selectedBuild.value) return
        verified.value = null
        verified.value = await client.verify(selectedBuild.value.build_id)
      }),
    resume: watchBuild,
    dispose: () => {
      disposed = true
      stop()
    }
  }
}
