<template>
  <div class="mx-auto w-full max-w-[1600px] space-y-6 px-5 py-6 lg:px-8">
    <div class="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-[#172536]">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div class="space-y-2">
          <UBadge color="primary" variant="soft">{{ t("knowledgeLab.proxyMode") }}</UBadge>
          <div class="flex items-center gap-3">
            <UIcon name="i-heroicons-book-open" class="text-primary-600 dark:text-primary-300" />
            <div>
              <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text058') }}</p>
              <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text059') }}</h1>
              <p class="text-sm text-gray-600 dark:text-gray-300">{{ t("knowledgeLab.description") }}</p>
            </div>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <UButton icon="i-heroicons-plus-circle" color="primary" variant="soft" :disabled="!canCreateSpace" @click="openCreateSpaceModal">{{ t("knowledgeSpaces.ui.createSpace") }}</UButton>
          <UButton icon="i-heroicons-magnifying-glass" color="neutral" variant="soft" @click="focusPlayground">{{ t("knowledgeLab.label0") }}</UButton>
          <UButton icon="i-heroicons-arrow-path" color="neutral" variant="ghost" :loading="providerLoading" @click="refreshKnowledgeState">{{ t('knowledgeLab.copy.text060') }}</UButton>
        </div>
      </div>
    </div>

    <UAlert
      v-if="providerError"
      icon="i-heroicons-exclamation-triangle"
      color="error"
      variant="soft"
      :title="t('knowledgeLab.copy.text061')"
      :description="providerError"
    />

    <UAlert v-if="catalogError" color="warning" variant="soft" :title="t('knowledgeLab.catalogUnavailable')" :description="catalogError" />
    <UAlert v-if="spacesError" color="error" variant="soft" :title="t('knowledgeSpaces.ui.loadFailed')" :description="spacesError" />

    <UAlert
      v-if="knowledgeNotice.message"
      :icon="knowledgeNotice.icon"
      :color="knowledgeNotice.color"
      variant="soft"
      :title="knowledgeNotice.title"
      :description="knowledgeNotice.message"
    />

    <ToastAlert
      v-model="toast.visible"
      :title="toast.title"
      :message="toast.message"
      :color="toast.color"
      :duration="toast.duration"
    />

    <UModal
      v-model:open="createSpaceOpen"
      :title="t('knowledgeLab.copy.text062')"
      :description="t('knowledgeLab.copy.text063')"
      :ui="{ content: 'w-[min(96vw,1040px)] max-w-none max-h-[calc(100dvh-2rem)] overflow-y-auto', body: 'p-6', footer: 'p-6 pt-0' }"
    >
      <template #body>
        <form id="knowledge-space-create-form" class="w-full space-y-6" @submit.prevent="createKnowledgeSpace">
          <section class="w-full rounded-lg border border-gray-200 bg-white dark:border-gray-800 dark:bg-slate-950">
            <div class="border-b border-gray-200 px-6 py-4 dark:border-gray-800">
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text064') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text065') }}</p>
            </div>
            <div class="grid gap-5 p-6 lg:grid-cols-[minmax(0,1.2fr)_minmax(280px,0.8fr)]">
              <UFormField :label="t('knowledgeLab.copy.text066')" required>
                <UInput v-model="createForm.spaceName" class="w-full" icon="i-heroicons-rectangle-stack" :placeholder="t('knowledgeLab.copy.text067')" :disabled="createLoading" />
              </UFormField>
              <UFormField :label="t('knowledgeLab.copy.text068')" required>
                <USelect v-model="createForm.departmentCode" :items="departmentSelectItems" class="w-full" :disabled="createLoading" />
              </UFormField>
            </div>
          </section>

          <section class="w-full rounded-lg border border-gray-200 bg-white dark:border-gray-800 dark:bg-slate-950">
            <div class="border-b border-gray-200 px-6 py-4 dark:border-gray-800">
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text069') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ selectedScene.description }}</p>
            </div>
            <div class="grid gap-5 p-6">
              <UFormField :label="t('knowledgeLab.copy.text070')" required>
                <USelectMenu
                  :model-value="createForm.sceneKey"
                  :items="knowledgeSceneItems"
                  value-key="value"
                  label-key="label"
                  :search-input="{ placeholder: t('knowledgeLab.copy.text071') }"
                  :portal="true"
                  class="w-full"
                  :ui="{ content: 'z-[80] min-w-[min(90vw,900px)] max-h-72 overflow-y-auto', item: 'whitespace-normal break-words' }"
                  :disabled="createLoading"
                  @update:model-value="onKnowledgeSceneChange"
                />
                <template #help>
                  <span class="text-xs text-gray-500 dark:text-gray-400">{{ selectedScene.description }}</span>
                </template>
              </UFormField>

              <UFormField :label="t('knowledgeLab.copy.text072')" required>
                <USelectMenu
                  :model-value="createForm.strategyPackageKey"
                  :items="strategyPackageItems"
                  value-key="value"
                  label-key="label"
                  :search-input="{ placeholder: t('knowledgeLab.copy.text073') }"
                  :portal="true"
                  class="w-full"
                  :ui="{ content: 'z-[80] min-w-[min(90vw,900px)] max-h-72 overflow-y-auto', item: 'whitespace-normal break-words' }"
                  :disabled="createLoading"
                  @update:model-value="onStrategyPackageChange"
                />
                <template #help>
                  <span class="text-xs text-gray-500 dark:text-gray-400">{{ strategyUseCase(selectedStrategyPackage) }}</span>
                </template>
              </UFormField>
              <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm text-gray-600 dark:border-gray-800 dark:bg-slate-900 dark:text-gray-300">
                <div class="font-medium text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text074') }}</div>
                <div class="mt-2">{{ t('knowledgeLab.copy.text075') }}{{ selectedScene.label }}</div>
                <div class="mt-1">{{ t('knowledgeLab.copy.text076') }}{{ strategyDisplayLabel(selectedStrategyPackage) }}</div>
                <div class="mt-1">{{ t('knowledgeLab.copy.text077') }}{{ strategyUseCase(selectedStrategyPackage) }}</div>
                <div class="mt-1">{{ t('knowledgeLab.copy.text078') }}{{ profileLabel(selectedStrategyPackage.recommended_profile_key) }}</div>
                <div class="mt-1">{{ t('knowledgeLab.copy.text079') }}{{ selectedScene.label }}</div>
                <div class="mt-1">{{ t('knowledgeLab.copy.text080') }}{{ strategyNotFor(selectedStrategyPackage) }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-slate-950">
                <UAlert color="info" variant="soft" :title="t('knowledgeLab.copy.text081')" />
                <div class="mt-4 flex flex-wrap items-center gap-2">
                  <span class="text-sm font-medium text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text082') }}</span>
                  <UBadge v-for="channel in enabledIndexChannels" :key="channel" color="neutral" variant="soft">{{ channelLabel(channel) }}</UBadge>
                  <span v-if="!enabledIndexChannels.length" class="text-sm text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text083') }}</span>
                </div>
                <div class="mt-3 grid grid-cols-1 gap-2 text-sm md:grid-cols-3">
                  <div class="text-gray-500 dark:text-gray-400">{{ t("knowledgeLab.label1") }}<span class="text-gray-900 dark:text-white">{{ createForm.ingestionProfileKey }}</span></div>
                  <div class="text-gray-500 dark:text-gray-400">{{ t("knowledgeLab.label2") }}<span class="text-gray-900 dark:text-white">{{ createForm.indexProfileKey }}</span></div>
                  <div class="text-gray-500 dark:text-gray-400">{{ t("knowledgeLab.label3") }}<span class="text-gray-900 dark:text-white">{{ createForm.ragProfileKey }}</span></div>
                </div>
              </div>
            </div>
          </section>

          <UAlert
            v-if="createError"
            icon="i-heroicons-x-circle"
            color="error"
            variant="soft"
            :title="t('knowledgeLab.copy.text084')"
            :description="createError"
          />
        </form>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="soft" :disabled="createLoading" @click="createSpaceOpen = false">{{ t('knowledgeLab.copy.text085') }}</UButton>
          <UButton type="submit" form="knowledge-space-create-form" icon="i-heroicons-plus-circle" :loading="createLoading">{{ t('knowledgeLab.copy.text086') }}</UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="ingestionOpen"
      :title="t('knowledgeLab.copy.text087')"
      :description="t('knowledgeLab.copy.text088')"
      :ui="{ content: 'w-[min(96vw,860px)] max-w-none', body: 'p-6', footer: 'p-6 pt-0' }"
    >
      <template #body>
        <form id="knowledge-ingestion-form" class="space-y-5" @submit.prevent="submitIngestion">
          <UAlert
            v-if="ingestionError"
            icon="i-heroicons-x-circle"
            color="error"
            variant="soft"
            :title="t('knowledgeLab.copy.text047')"
            :description="ingestionError"
          />

          <UFormField :label="t('knowledgeLab.copy.text089')" required>
            <USelect v-model="ingestionForm.spaceId" :items="spaceSelectItems" class="w-full" :disabled="ingestionLoading || !spaceSelectItems.length" />
          </UFormField>

          <div v-if="ingestionSpace.id" class="rounded-md border border-gray-200 bg-gray-50 p-3 text-sm dark:border-gray-800 dark:bg-slate-950">
            <div class="font-medium text-gray-900 dark:text-white">{{ ingestionSpace.spaceName || ingestionSpace.name }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ ingestionSpace.departmentCode || ingestionSpace.department || '-' }} · {{ ingestionSpace.statusLabel || ingestionSpace.status }}
            </div>
            <div class="mt-2 flex flex-wrap gap-2">
              <UBadge :color="modeColor" variant="soft">{{ providerModeLabel(providerState.mode || ingestionSpace.provider_mode || 'delegated') }}</UBadge>
              <UBadge color="neutral" variant="soft">{{ providerState.provider || ingestionSpace.provider || 'delegated' }}</UBadge>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-3">
            <UFormField :label="t('knowledgeLab.copy.text090')" required>
              <USelect v-model="ingestionSourceMethod" :items="ingestionSourceMethodItems" class="w-full" :disabled="ingestionLoading" />
            </UFormField>
            <UFormField :label="t('knowledgeLab.copy.text091')" required>
              <USelect v-model="ingestionForm.format" :items="ingestionFormatItems" class="w-full" :disabled="ingestionLoading" />
            </UFormField>
            <UFormField :label="t('knowledgeSpaces.ui.ingestionProfile')">
              <UInput v-model="ingestionForm.ingestionProfile" class="w-full" placeholder="p1_general" :disabled="ingestionLoading" />
            </UFormField>
          </div>

          <UFormField v-if="ingestionSourceMethod === 'upload'" :label="t('knowledgeLab.copy.text092')" required>
            <input
              type="file"
              accept=".pdf,.md,.markdown,.txt,.csv,.html,.htm"
              class="block w-full rounded-md border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 file:mr-3 file:rounded-md file:border-0 file:bg-primary-50 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-primary-700 hover:file:bg-primary-100 dark:border-gray-800 dark:bg-slate-950 dark:text-gray-200"
              :disabled="ingestionLoading"
              @change="onIngestionFileChange"
            />
            <template #help>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text093') }}</span>
            </template>
          </UFormField>

          <UFormField v-else :label="t('knowledgeLab.copy.text094')" required>
            <UInput v-model="ingestionForm.sourceUri" class="w-full" icon="i-heroicons-link" placeholder="https://..." :disabled="ingestionLoading" />
            <template #help>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text095') }}</span>
            </template>
          </UFormField>

          <UFormField :label="t('knowledgeLab.requestedBy')">
            <UInput v-model="ingestionForm.requestedBy" class="w-full" :disabled="ingestionLoading" />
          </UFormField>
        </form>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="soft" :disabled="ingestionLoading" @click="ingestionOpen = false">{{ t('knowledgeLab.copy.text085') }}</UButton>
          <UButton type="submit" form="knowledge-ingestion-form" icon="i-heroicons-arrow-up-tray" :loading="ingestionLoading" :disabled="!canSubmitIngestion">{{ t('knowledgeLab.copy.text096') }}</UButton>
        </div>
      </template>
    </UModal>

    <UModal
      v-model:open="resultOpen"
      :title="resultPanelTitle"
      :description="t('knowledgeLab.copy.text097')"
      :ui="{ content: 'max-w-5xl w-[92vw] mx-auto' }"
    >
      <template #body>
        <div class="max-h-[72vh] space-y-4 overflow-auto pr-1">
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-document-magnifying-glass" class="text-primary-600 dark:text-primary-300" />
              <span class="font-semibold text-gray-900 dark:text-white">{{ resultPanelTitle }}</span>
            </div>
            <UBadge color="neutral" variant="soft">{{ resultCountLabel }}</UBadge>
          </div>

          <UAlert
            v-if="searchError"
            icon="i-heroicons-x-circle"
            color="error"
            variant="soft"
            :title="resultErrorTitle"
            :description="searchError"
          />

          <UAlert
            v-if="!searchError && diagnosticsSummary && resultKind !== 'policy'"
            icon="i-heroicons-information-circle"
            color="info"
            variant="soft"
            :title="diagnosticsTitle"
            :description="diagnosticsSummary"
          />

          <div v-if="!searchError && resultKind === 'ingestions'" class="space-y-4">
            <div v-if="ingestionJobs.length" class="overflow-x-auto">
              <table class="min-w-[780px] divide-y divide-gray-200 text-sm dark:divide-gray-800">
                <thead>
                  <tr class="text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
                    <th class="px-3 py-2">{{ t('knowledgeLab.copy.text098') }}</th>
                    <th class="px-3 py-2">{{ t('knowledgeLab.copy.text099') }}</th>
                    <th class="px-3 py-2">{{ t("knowledgeLab.label4") }}</th>
                    <th class="px-3 py-2">{{ t('knowledgeLab.copy.text100') }}</th>
                    <th class="px-3 py-2">{{ t("knowledgeLab.label5") }}</th>
                    <th class="px-3 py-2">{{ t('knowledgeLab.copy.text101') }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
                  <tr v-for="(job, jobIndex) in pagedIngestionJobs" :key="job.jobId || job.id" class="hover:bg-gray-50 dark:hover:bg-slate-800/70">
                    <td class="px-3 py-3 font-mono text-xs text-gray-700 dark:text-gray-200">{{ t("knowledgeLab.taskNumber", { number: (ingestionPage - 1) * ingestionPageSize + jobIndex + 1 }) }}</td>
                    <td class="px-3 py-3">
                      <UBadge :color="ingestionStatusColor(job.status)" variant="soft">{{ job.status || '-' }}</UBadge>
                    </td>
                    <td class="px-3 py-3 text-gray-700 dark:text-gray-200">{{ job.chunkTotal ?? '-' }}</td>
                    <td class="px-3 py-3 text-gray-700 dark:text-gray-200">{{ formatPercent(job.chunkCoveragePct) }}</td>
                    <td class="px-3 py-3 text-gray-700 dark:text-gray-200">{{ formatPercent(job.embeddingSuccessPct) }}</td>
                    <td class="max-w-[260px] px-3 py-3 text-xs text-gray-500 dark:text-gray-400">{{ job.reason || job.errorCode || '-' }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <UAlert
              v-else
              icon="i-heroicons-inbox"
              color="neutral"
              variant="soft"
              :title="t('knowledgeLab.copy.text102')"
              :description="t('knowledgeLab.copy.text103')"
            />
            <div v-if="ingestionJobs.length" class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 pt-4 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400">
              <div>{{ t('knowledgeLab.copy.text104') }}{{ ingestionPageStart }}-{{ ingestionPageEnd }}{{ t('knowledgeLab.copy.text105') }}{{ ingestionJobs.length }}{{ t('knowledgeLab.copy.text106') }}</div>
              <div class="flex items-center gap-2">
                <span>{{ t('knowledgeLab.copy.text107') }}</span>
                <USelect v-model="ingestionPageSize" :items="ingestionPageSizeItems" class="w-24" />
                <UButton size="xs" color="neutral" variant="soft" :disabled="ingestionPage <= 1" @click="ingestionPage--">{{ t('knowledgeLab.copy.text108') }}</UButton>
                <span>{{ t('knowledgeLab.copy.text109') }}{{ ingestionPage }} / {{ ingestionPageCount }}{{ t('knowledgeLab.copy.text110') }}</span>
                <UButton size="xs" color="neutral" variant="soft" :disabled="ingestionPage >= ingestionPageCount" @click="ingestionPage++">{{ t('knowledgeLab.copy.text111') }}</UButton>
              </div>
            </div>
          </div>
          <div v-else-if="!searchError && resultKind === 'policy'" class="space-y-4">
            <div class="grid gap-4 md:grid-cols-2">
              <div class="rounded-md border border-gray-200 p-4 text-sm dark:border-gray-800">
                <div class="font-medium text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text112') }}</div>
                <div class="mt-3 space-y-2 text-gray-600 dark:text-gray-300">
                  <div>{{ t('knowledgeLab.copy.text113') }}{{ selectedSpace.spaceName || selectedSpace.name || '-' }}</div>
                  <div>{{ t('knowledgeLab.copy.text114') }}{{ selectedSpace.statusLabel || selectedSpace.status || '-' }}</div>
                  <div>{{ t('knowledgeLab.copy.text115') }}{{ selectedSpace.ingestionProfileKey || '-' }}</div>
                  <div>{{ t('knowledgeLab.copy.text116') }}{{ selectedSpace.indexProfileKey || '-' }}</div>
                  <div>RAG Profile：{{ selectedSpace.ragProfileKey || '-' }}</div>
                </div>
              </div>
              <div class="rounded-md border border-gray-200 p-4 text-sm dark:border-gray-800">
                <div class="font-medium text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text117') }}</div>
                <div class="mt-3 space-y-2 text-gray-600 dark:text-gray-300">
                  <div>{{ t('knowledgeLab.copy.text118') }}{{ strategyDisplayLabel(selectedPolicyPackage) }}</div>
                  <div>{{ t('knowledgeLab.copy.text078') }}{{ profileLabel(selectedPolicyPackage.recommended_profile_key) }}</div>
                  <div>{{ t('knowledgeLab.copy.text119') }}{{ selectedSpace.policyTemplateVersionId || '-' }}</div>
                  <div>{{ t('knowledgeLab.copy.text120') }}{{ providerModeLabel(providerState.mode || String(policyPayload?.provider_mode || 'delegated')) }}</div>
                </div>
              </div>
            </div>
            <div class="grid gap-4 md:grid-cols-3">
              <div class="rounded-md border border-gray-200 p-4 text-sm dark:border-gray-800">
                <div class="font-medium text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text121') }}</div>
                <div class="mt-3 flex flex-wrap gap-2">
                  <UBadge v-for="channel in selectedPolicyIndexChannels" :key="channel" color="primary" variant="soft">{{ channelLabel(channel) }}</UBadge>
                  <span v-if="!selectedPolicyIndexChannels.length" class="text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text083') }}</span>
                </div>
              </div>
              <div class="rounded-md border border-gray-200 p-4 text-sm dark:border-gray-800">
                <div class="font-medium text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text122') }}</div>
                <div class="mt-3 flex flex-wrap gap-2">
                  <UBadge v-for="item in selectedPolicyRuntimeDeps" :key="item" color="neutral" variant="soft">{{ item }}</UBadge>
                  <span v-if="!selectedPolicyRuntimeDeps.length" class="text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text123') }}</span>
                </div>
              </div>
              <div class="rounded-md border border-gray-200 p-4 text-sm dark:border-gray-800">
                <div class="font-medium text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text124') }}</div>
                <div class="mt-3 flex flex-wrap gap-2">
                  <UBadge v-for="item in selectedPolicyAssetDeps" :key="item" color="neutral" variant="soft">{{ item }}</UBadge>
                  <span v-if="!selectedPolicyAssetDeps.length" class="text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text123') }}</span>
                </div>
              </div>
            </div>
            <div class="rounded-md border border-gray-200 p-4 dark:border-gray-800">
              <div class="flex items-center justify-between gap-3">
                <div>
                  <div class="font-medium text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text125') }}</div>
                  <div class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text126') }}</div>
                </div>
                <UBadge color="neutral" variant="soft">{{ policyStrategies.length }}{{ t('knowledgeLab.copy.text106') }}</UBadge>
              </div>
              <div v-if="policyStrategies.length" class="mt-4 overflow-x-auto">
                <table class="min-w-[720px] divide-y divide-gray-200 text-sm dark:divide-gray-800">
                  <thead>
                    <tr class="text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
                      <th class="px-3 py-2">{{ t('knowledgeLab.copy.text127') }}</th>
                      <th class="px-3 py-2">{{ t('knowledgeLab.copy.text099') }}</th>
                      <th class="px-3 py-2">{{ t('knowledgeLab.copy.text128') }}</th>
                      <th class="px-3 py-2">{{ t('knowledgeLab.copy.text129') }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
                    <tr v-for="strategy in policyStrategies" :key="String(strategy.id || strategy.uuid || strategy.name || strategy.strategyId)" class="hover:bg-gray-50 dark:hover:bg-slate-800/70">
                      <td class="px-3 py-3 text-gray-700 dark:text-gray-200">{{ strategy.name || strategy.strategyName || strategy.strategyId || '-' }}</td>
                      <td class="px-3 py-3"><UBadge color="neutral" variant="soft">{{ strategy.status || '-' }}</UBadge></td>
                      <td class="px-3 py-3 text-gray-700 dark:text-gray-200">{{ strategy.weight ?? strategy.scoreWeight ?? '-' }}</td>
                      <td class="px-3 py-3 text-gray-500 dark:text-gray-400">{{ strategy.description || strategy.reason || '-' }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <UAlert
                v-else
                icon="i-heroicons-adjustments-horizontal"
                color="neutral"
                variant="soft"
                :title="t('knowledgeLab.copy.text130')"
                :description="t('knowledgeLab.copy.text131')"
                class="mt-4"
              />
            </div>
          </div>
          <div v-else-if="!searchError && result?.chunks?.length" class="space-y-3">
            <div
              v-for="chunk in result.chunks"
              :key="chunk.chunk_id"
              class="rounded-md border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-slate-950"
            >
              <div class="flex flex-wrap items-center justify-between gap-2">
                <div class="font-medium text-gray-900 dark:text-white">{{ chunk.citation?.title || t("knowledgeLab.unnamedDocument") }}</div>
                <UBadge color="success" variant="soft">{{ t('knowledgeLab.copy.text132') }}{{ formatScore(chunk.score) }}</UBadge>
              </div>
              <p class="mt-2 whitespace-pre-wrap text-sm text-gray-700 dark:text-gray-200">{{ chunk.text }}</p>
              <div class="mt-3 text-xs text-gray-500 dark:text-gray-400">
                {{ chunk.citation?.provider || result.provider }}
              </div>
            </div>
          </div>
          <div v-else-if="!searchError" class="py-8 text-sm text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text133') }}</div>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end">
          <UButton color="neutral" variant="soft" @click="resultOpen = false">{{ t('knowledgeLab.copy.text134') }}</UButton>
        </div>
      </template>
    </UModal>

    <div
      v-if="contractGaps.length"
      class="flex items-start gap-3 rounded-md border border-sky-100 bg-sky-50 px-4 py-3 text-sm text-sky-900 dark:border-sky-900/40 dark:bg-sky-950/30 dark:text-sky-100"
    >
      <UIcon name="i-heroicons-information-circle" class="mt-0.5 shrink-0 text-sky-600 dark:text-sky-300" />
      <div class="min-w-0">
        <div class="font-medium">{{ t('knowledgeLab.copy.text135') }}</div>
        <div class="mt-1 text-sky-800/80 dark:text-sky-100/80">{{ t('knowledgeLab.copy.text136') }}</div>
      </div>
    </div>

    <div class="space-y-6">
      <div class="space-y-6">
        <UCard>
          <template #header>
            <div class="space-y-4">
              <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text089') }}</h2>
                <div class="flex flex-wrap items-center gap-2">
                  <UBadge :color="modeColor" variant="soft">{{ providerModeLabel(providerState.mode || 'delegated') }}</UBadge>
                  <UBadge color="neutral" variant="soft">{{ spaces.length }}{{ t('knowledgeLab.copy.text137') }}</UBadge>
                </div>
              </div>
              <div class="grid gap-3 lg:grid-cols-[minmax(220px,1fr)_180px_180px]">
                <UInput v-model="filters.keyword" icon="i-heroicons-magnifying-glass" :placeholder="t('knowledgeLab.copy.text138')" />
                <USelect v-model="filters.department" :items="departmentItems" class="w-full" />
                <USelect v-model="filters.status" :items="statusItems" class="w-full" />
              </div>
            </div>
          </template>

          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-gray-800">
              <thead>
                <tr class="text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  <th class="w-[280px] px-4 py-3">{{ t('knowledgeLab.copy.text139') }}</th>
                  <th class="w-[140px] px-4 py-3">{{ t('knowledgeLab.copy.text140') }}</th>
                  <th class="w-[120px] px-4 py-3">{{ t('knowledgeLab.copy.text099') }}</th>
                  <th class="w-[260px] px-4 py-3 text-right">{{ t('knowledgeLab.copy.text141') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
                <tr
                  v-for="space in filteredSpaces"
                  :key="space.id"
                  :class="[
                    'cursor-pointer transition hover:bg-gray-50 dark:hover:bg-slate-800/70',
                    selectedSpaceId === space.id ? 'bg-primary-50/70 dark:bg-primary-950/20' : ''
                  ]"
                  @click="selectSpace(space.id)"
                >
                  <td class="px-4 py-4">
                    <div class="flex items-start gap-3">
                      <div class="flex h-9 w-9 items-center justify-center rounded-md bg-primary-50 text-primary-600 dark:bg-primary-950/40 dark:text-primary-300">
                        <UIcon :name="space.icon" />
                      </div>
                      <div>
                        <div class="flex flex-wrap items-center gap-2">
                          <span class="font-semibold text-gray-900 dark:text-white">{{ space.spaceName || space.name }}</span>
                          <UBadge v-if="space.department" color="neutral" variant="soft">{{ space.department }}</UBadge>
                        </div>
                      </div>
                    </div>
                  </td>
                  <td class="px-4 py-4">
                    <div class="space-y-1">
                      <div class="font-medium text-gray-900 dark:text-white">{{ space.departmentCode || space.department || '-' }}</div>
                      <UBadge color="info" variant="soft">{{ spaceTypeLabel(space) }}</UBadge>
                    </div>
                  </td>
                  <td class="px-4 py-4">
                    <UBadge :color="space.statusColor" variant="soft">{{ space.statusLabel }}</UBadge>
                  </td>
                  <td class="px-4 py-4">
                    <div class="flex min-w-max flex-wrap justify-end gap-2">
                      <UButton size="xs" color="success" variant="soft" icon="i-heroicons-arrow-up-tray" @click.stop="openIngestionModal(space)">{{ t('knowledgeLab.copy.text142') }}</UButton>
                      <UButton size="xs" color="primary" variant="soft" icon="i-heroicons-link" @click.stop="showDataSource(space.id)">{{ t('knowledgeLab.copy.text143') }}</UButton>
                      <UButton size="xs" color="neutral" variant="soft" icon="i-heroicons-clipboard-document-list" @click.stop="showIngestion(space.id)">{{ t('knowledgeLab.copy.text030') }}</UButton>
                      <UButton size="xs" color="neutral" variant="soft" icon="i-heroicons-adjustments-horizontal" @click.stop="showPolicy(space.id)">{{ t('knowledgeLab.copy.text127') }}</UButton>
                      <UButton v-if="space.status !== 'retired'" size="xs" color="error" variant="soft" icon="i-heroicons-archive-box" :loading="retiringSpaceId === space.id" @click.stop="retireSpace(space)">{{ t('knowledgeLab.copy.text144') }}</UButton>
                      <UButton size="xs" color="error" variant="solid" icon="i-heroicons-trash" :loading="deletingSpaceId === space.id" :disabled="space.status !== 'retired'" @click.stop="deleteSpace(space)">{{ t('knowledgeLab.copy.text145') }}</UButton>
                      <UButton size="xs" color="neutral" variant="soft" icon="i-heroicons-magnifying-glass" @click.stop="focusPlayground(space.id)">{{ t("knowledgeLab.label0") }}</UButton>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

            <div v-if="!providerLoading && !spacesError && !filteredSpaces.length" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
              {{ spaces.length ? t('knowledgeLab.copy.text146') : t('knowledgeLab.copy.text147') }}
            </div>
        </UCard>

      </div>

      <div class="space-y-6">
        <UCard>
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-command-line" class="text-primary-600 dark:text-primary-300" />
              <span class="font-semibold text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text148') }}</span>
            </div>
          </template>

          <form ref="playgroundFormRef" class="space-y-4" @submit.prevent="runSearch">
            <UFormField :label="t('knowledgeLab.copy.text149')">
              <USelect v-model="selectedSpaceId" :items="spaceSelectItems" class="w-full" :disabled="!spaceSelectItems.length" :placeholder="t('knowledgeLab.copy.text150')" />
            </UFormField>
            <div v-if="selectedSpaceId" class="rounded-md border border-gray-200 bg-gray-50 p-3 text-sm dark:border-gray-800 dark:bg-slate-950">
              <div class="font-medium text-gray-900 dark:text-white">{{ selectedSpace.spaceName || selectedSpace.name }}</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ selectedSpace.departmentCode || selectedSpace.department || '-' }} · {{ selectedSpace.statusLabel || selectedSpace.status }}
              </div>
              <div class="mt-2 flex flex-wrap gap-2">
                <UBadge :color="modeColor" variant="soft">{{ providerModeLabel(providerState.mode || selectedSpace.provider_mode || 'delegated') }}</UBadge>
                <UBadge color="neutral" variant="soft">{{ providerState.provider || selectedSpace.provider || 'delegated' }}</UBadge>
              </div>
            </div>
            <UAlert
              v-else
              icon="i-heroicons-information-circle"
              color="neutral"
              variant="soft"
              :title="t('knowledgeLab.copy.text151')"
              :description="t('knowledgeLab.selectSpace')"
            />
            <div class="grid gap-3 sm:grid-cols-2">
              <UFormField :label="t('knowledgeLab.copy.text070')">
                <USelect v-model="form.space_type" :items="knowledgeTypeItems" class="w-full" />
              </UFormField>
              <UFormField :label="t('knowledgeLab.copy.text152')">
                <USelect v-model="form.provider_mode" :items="providerModeItems" class="w-full" disabled />
              </UFormField>
              <UFormField :label="t('knowledgeLab.copy.text153')">
                <UInput v-model.number="form.limit" type="number" min="1" max="20" />
              </UFormField>
            </div>
            <UFormField :label="t('knowledgeLab.copy.text154')" required>
              <UTextarea v-model="form.query" :rows="4" :placeholder="t('knowledgeLab.copy.text155')" />
            </UFormField>

            <div class="flex justify-end gap-2">
              <UButton type="submit" icon="i-heroicons-play" :loading="searchLoading" :disabled="!form.query.trim() || !activeSpaceId || !!spacesError">{{ t('knowledgeLab.copy.text156') }}</UButton>
            </div>
          </form>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between">
              <span class="font-semibold text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text157') }}</span>
              <UBadge :color="healthColor" variant="soft">{{ healthLabel(capabilities.health) }}</UBadge>
            </div>
          </template>
          <div class="grid grid-cols-2 gap-3 text-sm">
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t("knowledgeLab.label6") }}</div>
              <div class="font-semibold text-gray-900 dark:text-white">{{ providerState.provider || '-' }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text158') }}</div>
              <div class="font-semibold text-gray-900 dark:text-white">{{ providerState.production ? t('knowledgeLab.copy.text159') : t('knowledgeLab.copy.text160') }}</div>
            </div>
          </div>
          <div class="mt-4 flex flex-wrap gap-2">
            <UBadge v-for="op in capabilities.operations || []" :key="op" color="info" variant="soft">{{ op }}</UBadge>
            <span v-if="!capabilities.operations?.length" class="text-sm text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text161') }}</span>
          </div>
          <div v-if="selectedSpace.contract_gaps?.length" class="mt-4 space-y-2">
            <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text162') }}</div>
            <div v-for="gap in selectedSpace.contract_gaps" :key="gap" class="rounded-md bg-gray-50 p-2 text-xs text-gray-600 dark:bg-slate-950 dark:text-gray-300">
              {{ gap }}
            </div>
          </div>
        </UCard>

        <UCard>
          <template #header>
            <span class="font-semibold text-gray-900 dark:text-white">{{ t('knowledgeLab.copy.text163') }}</span>
          </template>
          <div class="space-y-4">
            <div class="space-y-2">
              <div v-for="citation in result?.citations || []" :key="`${citation.document_id}:${citation.chunk_id}`" class="rounded-md bg-gray-50 p-3 text-sm dark:bg-slate-950">
                <div class="font-medium text-gray-900 dark:text-white">{{ citation.title }}</div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ citation.version || '-' }}</div>
              </div>
              <div v-if="!result?.citations?.length" class="text-sm text-gray-500 dark:text-gray-400">{{ t('knowledgeLab.copy.text164') }}</div>
            </div>
            <pre class="max-h-56 overflow-auto rounded-md bg-gray-950 p-3 text-xs text-gray-100">{{ diagnosticsText }}</pre>
          </div>
        </UCard>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from "vue";
import ToastAlert from "~/components/ToastAlert.vue";
import { apiPost, useApiClient } from "~/composables/api/_client";

type ProviderResponse = {
  provider?: string;
  mode?: string;
  production?: boolean;
  capabilities?: {
    provider?: string;
    mode?: string;
    operations?: string[];
    health?: string;
    limits?: Record<string, unknown>;
  };
};

type SearchResult = {
  provider?: string;
  total?: number;
  chunks?: Array<{
    chunk_id?: string;
    document_id?: string;
    space_id?: string;
    text?: string;
    score?: number;
    citation?: {
      title?: string;
      document_id?: string;
      chunk_id?: string;
      provider?: string;
      version?: string;
    };
  }>;
  citations?: Array<{
    title?: string;
    document_id?: string;
    chunk_id?: string;
    provider?: string;
    version?: string;
  }>;
  diagnostics?: Record<string, unknown>;
};

type IngestionJobRecord = {
  id?: string;
  jobId?: string;
  status?: string;
  retryCount?: number;
  errorCode?: string;
  reason?: string;
  chunkTotal?: number;
  chunkCoveragePct?: number;
  embeddingSuccessPct?: number;
  maskingCoveragePct?: number;
};

type SpaceRow = {
  id: string;
  name: string;
  spaceId?: string;
  spaceName?: string;
  tenant_uuid?: string;
  department: string;
  departmentCode?: string;
  type: string;
  typeLabel?: string;
  type_label?: string;
  status: string;
  statusLabel?: string;
  status_label?: string;
  statusColor?: "success" | "warning" | "neutral" | "error" | "info";
  status_color?: "success" | "warning" | "neutral" | "error" | "info";
  provider?: string;
  provider_mode?: string;
  policyTemplateVersionId?: string;
  ingestionProfileKey?: string;
  indexProfileKey?: string;
  ragProfileKey?: string;
  featureFlags?: string[];
  quotas?: Record<string, unknown>;
  contract_gaps?: string[];
  icon?: string;
};

type StrategyDependency = {
  index?: string[];
  runtime?: string[];
  assets?: string[];
};

type KnowledgeScenePreset = {
  key: string;
  label: string;
  description?: string;
  default_strategy_package?: string;
  default_bundle?: string;
};

type StrategyPackagePreset = {
  key: string;
  label: string;
  display_label?: string;
  summary?: string;
  use_case?: string;
  not_for?: string;
  recommended_profile_key?: string;
  dependencies?: StrategyDependency;
};

type StrategyBundlePreset = {
  key: string;
  label: string;
  description?: string;
};

type KnowledgeCatalog = {
  version?: string;
  source?: string;
  scenes?: KnowledgeScenePreset[];
  strategy_packages?: StrategyPackagePreset[];
  strategy_bundles?: StrategyBundlePreset[];
  metadata?: Record<string, unknown>;
};

const { get, post, delete: del } = useApiClient();
const route = useRoute();
const { t, te } = useI18n();

const providerLoading = ref(false);
const searchLoading = ref(false);
const createLoading = ref(false);
const ingestionLoading = ref(false);
const retiringSpaceId = ref("");
const deletingSpaceId = ref("");
const providerError = ref("");
const catalogError = ref("");
const spacesError = ref("");
const searchError = ref("");
const createError = ref("");
const ingestionError = ref("");
const diagnosticsTitle = ref(t('knowledgeLab.copy.text001'));
const providerState = ref<ProviderResponse>({ mode: "delegated" });
const knowledgeCatalog = ref<KnowledgeCatalog>({});
const canCreateSpace = computed(() => !catalogError.value && !!knowledgeCatalog.value.scenes?.length && !!knowledgeCatalog.value.strategy_packages?.length);
const result = ref<SearchResult | null>(null);
const ingestionJobs = ref<IngestionJobRecord[]>([]);
const policyPayload = ref<Record<string, any> | null>(null);
const selectedSpaceId = ref("");
const playgroundFormRef = ref<HTMLFormElement | null>(null);
const contractGaps = ref<string[]>([]);
const createSpaceOpen = ref(false);
const ingestionOpen = ref(false);
const resultOpen = ref(false);
const resultKind = ref<"diagnostics" | "ingestions" | "policy" | "search">("diagnostics");
const ingestionPage = ref(1);
const ingestionPageSize = ref(10);
const selectedIngestionFile = ref<File | null>(null);
const ingestionSourceMethod = ref<"upload" | "url">("upload");
const toast = reactive({
  visible: false,
  title: "",
  message: "",
  color: "success" as "primary" | "secondary" | "success" | "info" | "warning" | "error" | "neutral",
  duration: 4500,
});
const knowledgeNotice = reactive({
  title: "",
  message: "",
  color: "info" as "primary" | "secondary" | "success" | "info" | "warning" | "error" | "neutral",
  icon: "i-heroicons-information-circle",
});

const form = reactive({
  query: "",
  space_type: "tenant",
  provider_mode: "delegated",
  limit: 5,
});

const filters = reactive({
  keyword: "",
  department: "all",
  status: "all",
});

const createForm = reactive({
  spaceName: "",
  departmentCode: "",
  sceneKey: "product_specs",
  strategyPackageKey: "H_fusion",
  ingestionProfileKey: "p1_general",
  indexProfileKey: "p1_general",
  ragProfileKey: "p1_general",
  featureFlags: ["rag.strategy_package:H_fusion", "rag.scene:product_specs", "rag.bundle:p1_general"],
  cpuCores: 4,
  storageGb: 200,
  ingestionConcurrency: 2,
});

const ingestionForm = reactive({
  spaceId: "",
  format: "markdown",
  sourceUri: "",
  ingestionProfile: "p1_general",
  requestedBy: "plugin-knowledge-lab",
});

const spaces = ref<SpaceRow[]>([]);

const profileLabels: Record<string, string> = {
  p0_basic: t('knowledgeLab.copy.text002'),
  p1_general: t('knowledgeLab.copy.text003'),
  p2_high_accuracy: t('knowledgeLab.copy.text004'),
  p3_kg_strong: t('knowledgeLab.copy.text005'),
};

type IndexChannel = "dense" | "sparse" | "hier" | "kg" | "time" | "structured";
const knowledgeScenesByKey = computed(() => Object.fromEntries((knowledgeCatalog.value.scenes || []).map((scene) => [scene.key, scene])));
const strategyPackagesByKey = computed(() => Object.fromEntries((knowledgeCatalog.value.strategy_packages || []).map((pkg) => [pkg.key, pkg])));
const knowledgeSceneItems = computed(() => (knowledgeCatalog.value.scenes || []).map((scene) => ({ label: scene.label || scene.key, value: scene.key })));
const strategyPackageItems = computed(() => (knowledgeCatalog.value.strategy_packages || []).map((pkg) => ({ label: `${strategyDisplayLabel(pkg)} · ${pkg.summary || pkg.label || pkg.key}`, value: pkg.key })));
const selectedScene = computed(() => knowledgeScenesByKey.value[createForm.sceneKey] || (knowledgeCatalog.value.scenes || [])[0] || { key: "", label: "", description: "" });
const selectedStrategyPackage = computed(() => strategyPackagesByKey.value[createForm.strategyPackageKey] || (knowledgeCatalog.value.strategy_packages || [])[0] || { key: "", label: "", recommended_profile_key: "", dependencies: { index: [] } });

function profileLabel(key?: string) {
  return profileLabels[key || ""] || key || "-";
}

function normalizeStrategyPackageKey(value: unknown) {
  if (typeof value === "string") return value;
  if (value && typeof value === "object" && "value" in value) {
    return String((value as { value?: unknown }).value || "");
  }
  return "";
}

function normalizeSceneKey(value: unknown) {
  if (typeof value === "string") return value;
  if (value && typeof value === "object" && "value" in value) {
    return String((value as { value?: unknown }).value || "");
  }
  return "";
}

function setCreateStrategyPackage(value: unknown, sceneKey?: string) {
  const key = normalizeStrategyPackageKey(value);
  const packageKey = strategyPackagesByKey.value[key] ? key : selectedStrategyPackage.value.key;
  const pkg = strategyPackagesByKey.value[packageKey] || selectedStrategyPackage.value;
  const profileKey = pkg.recommended_profile_key || "p1_general";
  const resolvedSceneKey = sceneKey && knowledgeScenesByKey.value[sceneKey] ? sceneKey : createForm.sceneKey;
  createForm.strategyPackageKey = packageKey;
  createForm.ingestionProfileKey = profileKey;
  createForm.indexProfileKey = profileKey;
  createForm.ragProfileKey = profileKey;
  createForm.featureFlags = [
    "rag.strategy_package:" + packageKey,
    "rag.scene:" + resolvedSceneKey,
    "rag.bundle:" + profileKey,
  ];
  if (resolvedSceneKey === "custom_expert") {
    createForm.featureFlags.push("rag.guided");
  }
}

function setCreateKnowledgeScene(value: unknown) {
  const key = normalizeSceneKey(value);
  const sceneKey = knowledgeScenesByKey.value[key] ? key : selectedScene.value.key;
  createForm.sceneKey = sceneKey;
  setCreateStrategyPackage(knowledgeScenesByKey.value[sceneKey]?.default_strategy_package || "H_fusion", sceneKey);
}

function onKnowledgeSceneChange(value: unknown) {
  setCreateKnowledgeScene(value);
}

function onStrategyPackageChange(value: unknown) {
  setCreateStrategyPackage(value);
}

const enabledIndexChannels = computed<IndexChannel[]>(() => {
  const keys = new Set(selectedStrategyPackage.value.dependencies?.index || []);
  const out: IndexChannel[] = [];
  const push = (key: string, channel: IndexChannel) => {
    if (keys.has(key)) out.push(channel);
  };
  push("index.dense", "dense");
  push("index.sparse", "sparse");
  push("index.hier", "hier");
  push("index.kg", "kg");
  push("index.time_fields", "time");
  push("index.structured_fields", "structured");
  return out;
});

function channelLabel(channel: string) {
  switch (channel) {
    case "dense": return "Dense";
    case "sparse": return "Sparse(BM25)";
    case "hier": return "Hier";
    case "kg": return "KG";
    case "time": return "Time";
    case "structured": return "Structured";
    default: return channel;
  }
}

function strategyDisplayLabel(pkg: StrategyPackagePreset) {
  return pkg.display_label || pkg.label || pkg.key || "-";
}

function strategyUseCase(pkg: StrategyPackagePreset) {
  return pkg.use_case || pkg.summary || pkg.label || "-";
}

function strategyNotFor(pkg: StrategyPackagePreset) {
  return pkg.not_for || t('knowledgeLab.copy.text008');
}

const spaceTypeItems = [
  { label: t('knowledgeLab.copy.text009'), value: "all" },
  { label: t('knowledgeLab.copy.text010'), value: "tenant" },
  { label: t('knowledgeLab.copy.text011'), value: "plugin" },
  { label: t('knowledgeLab.copy.text012'), value: "public" },
  { label: t('knowledgeLab.copy.text013'), value: "agent" },
  { label: t('knowledgeLab.copy.text014'), value: "platform" },
];

const knowledgeTypeItems = spaceTypeItems.filter((item) => item.value !== "all");
const statusItems = [
  { label: t('knowledgeLab.copy.text015'), value: "all" },
  { label: t('knowledgeLab.copy.text016'), value: "active" },
  { label: t('knowledgeLab.copy.text017'), value: "pending_iam" },
  { label: t('knowledgeLab.copy.text018'), value: "retired" },
  { label: t('knowledgeLab.copy.text019'), value: "pending" },
  { label: t('knowledgeLab.copy.text020'), value: "unavailable" },
];
const providerModeItems = computed(() => [{ label: t("knowledgeLab.proxyMode"), value: "delegated" }]);
const ingestionFormatItems = [
  { label: "Markdown", value: "markdown" },
  { label: "PDF", value: "pdf" },
  { label: "Text", value: "text" },
  { label: "HTML", value: "html" },
  { label: "CSV", value: "csv" },
];
const ingestionSourceMethodItems = [
  { label: t('knowledgeLab.copy.text024'), value: "upload" },
  { label: t('knowledgeLab.copy.text025'), value: "url" },
];
const ingestionPageSizeItems = [
  { label: "5", value: 5 },
  { label: "10", value: 10 },
  { label: "20", value: 20 },
];

const departmentSelectItems = computed(() => {
  const departments = Array.from(new Set(spaces.value.map((space) => space.departmentCode || space.department).filter(Boolean))).sort();
  return departments.map((department) => ({ label: department, value: department }));
});

async function openCreateSpaceModal() {
  createError.value = "";
  if (!knowledgeCatalog.value.scenes?.length || !knowledgeCatalog.value.strategy_packages?.length) {
    await loadKnowledgeCatalog();
  }
  setCreateStrategyPackage(createForm.strategyPackageKey);
  createSpaceOpen.value = true;
}

async function createKnowledgeSpace() {
  createLoading.value = true;
  createError.value = "";
  try {
    if (!createForm.spaceName.trim()) {
      createError.value = t('knowledgeLab.copy.text026');
      return;
    }
    if (!createForm.departmentCode.trim()) {
      createError.value = t('knowledgeLab.copy.text027');
      return;
    }
    const payload: Record<string, any> = {
      spaceName: createForm.spaceName.trim(),
      departmentCode: createForm.departmentCode.trim(),
      visibility: "tenant",
      description: "",
      policyTemplateVersionId: "default-v1",
      ingestionProfileKey: createForm.ingestionProfileKey,
      indexProfileKey: createForm.indexProfileKey,
      ragProfileKey: createForm.ragProfileKey,
      featureFlags: [...createForm.featureFlags],
      cpuCores: createForm.cpuCores,
      storageGb: createForm.storageGb,
      ingestionConcurrency: createForm.ingestionConcurrency,
      requestedBy: "plugin-knowledge-lab",
    };
    const created = unwrap<Record<string, any>>(await post("/admin/runtime/knowledge-lab/spaces", payload));
    createSpaceOpen.value = false;
    await loadSpaces();
    const createdID = String(created?.spaceId || created?.uuid || "");
    if (createdID) {
      selectedSpaceId.value = createdID;
    }
  } catch (error: any) {
    if (isConflictError(error)) {
      await loadSpaces();
      const existing = findExistingSpace(createForm.spaceName, createForm.departmentCode);
      if (existing) {
        selectedSpaceId.value = existing.id;
      }
      createError.value = t('knowledgeLab.copy.text029');
      return;
    }
    createError.value = errorMessage(error);
  } finally {
    createLoading.value = false;
  }
}

async function retireSpace(space: SpaceRow) {
  const spaceID = String(space.id || space.spaceId || "").trim();
  if (!spaceID) return;
  const name = String(space.spaceName || space.name || t("knowledgeLab.unnamedSpace"));
  const ok = window.confirm(t("knowledgeLab.archiveConfirm", { name }));
  if (!ok) return;
  retiringSpaceId.value = spaceID;
  providerError.value = "";
  try {
    await post(`/admin/runtime/knowledge-lab/spaces/${encodeURIComponent(spaceID)}/retire`, {
      reason: "plugin knowledge lab cleanup",
      requestedBy: "plugin-knowledge-lab",
      dropVectors: false,
    });
    await loadSpaces();
    if (selectedSpaceId.value === spaceID && spaces.value.length && !spaces.value.some((item) => item.id === spaceID && item.status !== "retired")) {
      selectedSpaceId.value = spaces.value.find((item) => item.status !== "retired")?.id || spaces.value[0]?.id || "";
    }
  } catch (error: any) {
    providerError.value = errorMessage(error);
  } finally {
    retiringSpaceId.value = "";
  }
}

async function deleteSpace(space: SpaceRow) {
  const spaceID = String(space.id || space.spaceId || "").trim();
  if (!spaceID) return;
  const name = String(space.spaceName || space.name || t("knowledgeLab.unnamedSpace"));
  const ok = window.confirm(t("knowledgeLab.deleteConfirm", { name }));
  if (!ok) return;
  deletingSpaceId.value = spaceID;
  providerError.value = "";
  try {
    await del(`/admin/runtime/knowledge-lab/spaces/${encodeURIComponent(spaceID)}`, {
      body: JSON.stringify({
        requestedBy: "plugin-knowledge-lab",
        force: false,
        dropVectors: true,
      }),
      headers: {
        "Content-Type": "application/json",
      },
    });
    await loadSpaces();
    if (selectedSpaceId.value === spaceID) {
      selectedSpaceId.value = spaces.value[0]?.id || "";
    }
  } catch (error: any) {
    providerError.value = errorMessage(error);
  } finally {
    deletingSpaceId.value = "";
  }
}

const capabilities = computed(() => providerState.value.capabilities || {});
const resultPanelTitle = computed(() => {
  if (resultKind.value === "ingestions") return t('knowledgeLab.copy.text030');
  if (resultKind.value === "policy") return t('knowledgeLab.copy.text031');
  if (resultKind.value === "search") return t('knowledgeLab.copy.text032');
  return diagnosticsTitle.value || t('knowledgeLab.copy.text001');
});
const resultErrorTitle = computed(() => {
  if (resultKind.value === "ingestions") return t('knowledgeLab.copy.text033');
  if (resultKind.value === "policy") return t('knowledgeLab.copy.text034');
  if (resultKind.value === "search") return t('knowledgeLab.copy.text035');
  return t('knowledgeLab.copy.text036');
});
const resultCountLabel = computed(() => {
  if (resultKind.value === "ingestions") return t("knowledgeLab.count", { count: ingestionJobs.value.length });
  if (resultKind.value === "policy") return t("knowledgeLab.count", { count: policyStrategies.value.length });
  return t("knowledgeLab.count", { count: result.value?.total ?? result.value?.chunks?.length ?? 0 });
});
const modeColor = computed(() => {
  const mode = providerState.value.mode;
  if (mode === "delegated") return "primary";
  if (mode === "local") return "warning";
  if (mode === "mock") return "neutral";
  return "info";
});
const healthColor = computed(() => capabilities.value.health === "ready" ? "success" : "warning");
const diagnosticsText = computed(() => JSON.stringify(result.value?.diagnostics || {}, null, 2));
const diagnosticsSummary = computed(() => summarizeDiagnostics(result.value?.diagnostics));
const ingestionPageCount = computed(() => Math.max(1, Math.ceil(ingestionJobs.value.length / ingestionPageSize.value)));
const pagedIngestionJobs = computed(() => {
  const page = Math.min(Math.max(ingestionPage.value, 1), ingestionPageCount.value);
  const start = (page - 1) * ingestionPageSize.value;
  return ingestionJobs.value.slice(start, start + ingestionPageSize.value);
});
const ingestionPageStart = computed(() => ingestionJobs.value.length ? (Math.min(ingestionPage.value, ingestionPageCount.value) - 1) * ingestionPageSize.value + 1 : 0);
const ingestionPageEnd = computed(() => Math.min(ingestionPageStart.value + ingestionPageSize.value - 1, ingestionJobs.value.length));
const policyStrategies = computed(() => normalizePolicyStrategies(policyPayload.value));
const selectedPolicyPackage = computed(() => {
  const packageKey = inferStrategyPackageKey(selectedSpace.value);
  return strategyPackagesByKey.value[packageKey] || selectedStrategyPackage.value;
});
const selectedPolicyIndexChannels = computed<IndexChannel[]>(() => indexChannelsFromDependencies(selectedPolicyPackage.value.dependencies));
const selectedPolicyRuntimeDeps = computed(() => selectedPolicyPackage.value.dependencies?.runtime || []);
const selectedPolicyAssetDeps = computed(() => selectedPolicyPackage.value.dependencies?.assets || []);
const departmentItems = computed(() => {
  const departments = Array.from(new Set(spaces.value.map((space) => space.departmentCode || space.department).filter(Boolean))).sort();
  return [{ label: t('knowledgeLab.copy.text037'), value: "all" }, ...departments.map((department) => ({ label: department, value: department }))];
});
const spaceSelectItems = computed(() => spaces.value.map((space) => ({ label: `${space.spaceName || space.name} · ${spaceTypeLabel(space)}`, value: space.id })));
const selectedSpace = computed(() => spaces.value.find((space) => space.id === selectedSpaceId.value) || ({} as SpaceRow));
const ingestionSpace = computed(() => spaces.value.find((space) => space.id === ingestionForm.spaceId) || ({} as SpaceRow));
const canSubmitIngestion = computed(() => {
  if (!ingestionForm.spaceId) return false;
  if (ingestionSourceMethod.value === "upload") return Boolean(selectedIngestionFile.value);
  return Boolean(ingestionForm.sourceUri.trim());
});
const activeSpaceId = computed(() => selectedSpaceId.value);
const filteredSpaces = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase();
  return spaces.value.filter((space) => {
    const department = space.departmentCode || space.department;
    const matchKeyword = !keyword || `${space.spaceName || space.name} ${space.id} ${department} ${space.policyTemplateVersionId || ""} ${space.ingestionProfileKey || ""} ${space.indexProfileKey || ""} ${space.ragProfileKey || ""}`.toLowerCase().includes(keyword);
    const matchDepartment = filters.department === "all" || department === filters.department;
    const matchStatus = filters.status === "all" || space.status === filters.status;
    return matchKeyword && matchDepartment && matchStatus;
  });
});

function indexChannelsFromDependencies(dependencies?: StrategyDependency): IndexChannel[] {
  const keys = new Set(dependencies?.index || []);
  const channels: IndexChannel[] = [];
  const push = (key: string, channel: IndexChannel) => {
    if (keys.has(key)) channels.push(channel);
  };
  push("index.dense", "dense");
  push("index.sparse", "sparse");
  push("index.hier", "hier");
  push("index.kg", "kg");
  push("index.time_fields", "time");
  push("index.structured_fields", "structured");
  return channels;
}

function inferStrategyPackageKey(space: SpaceRow) {
  const flags = (space.featureFlags || []).map((flag) => String(flag || "").trim());
  const fromFlag = flags.find((flag) => flag.toLowerCase().startsWith("rag.strategy_package:"))?.slice("rag.strategy_package:".length);
  if (fromFlag && strategyPackagesByKey.value[fromFlag]) return fromFlag;
  const profile = space.ragProfileKey || space.indexProfileKey || space.ingestionProfileKey;
  if (profile === "p0_basic" && strategyPackagesByKey.value.A_simple) return "A_simple";
  if (profile === "p2_high_accuracy" && strategyPackagesByKey.value.O_crag) return "O_crag";
  if (profile === "p3_kg_strong" && strategyPackagesByKey.value.K_kg) return "K_kg";
  return strategyPackagesByKey.value.H_fusion ? "H_fusion" : selectedStrategyPackage.value.key;
}

watch(selectedSpace, (space) => {
  form.space_type = space.type === "platform" ? "tenant" : space.type;
});

watch(
  () => providerState.value.mode,
  (mode) => {
    form.provider_mode = mode || "delegated";
  },
);

watch([ingestionPageSize, ingestionJobs], () => {
  ingestionPage.value = Math.min(Math.max(ingestionPage.value, 1), ingestionPageCount.value);
});

function unwrap<T>(payload: any): T {
  if (payload && typeof payload === "object" && "data" in payload) {
    return payload.data as T;
  }
  return payload as T;
}

function errorMessage(error: any) {
  const message = String(error?.data?.error?.message || error?.data?.error || error?.message || "");
  if (te(message) && message !== "knowledgeLab.gatewayFailed") return t(message);
  const code = String(error?.data?.code || error?.data?.error?.code || error?.statusCode || error?.status || "");
  const trace = String(error?.data?.trace_id || "");
  return t("knowledgeLab.requestFailed", { code: code || "UNKNOWN" }) + (trace ? ` (${trace})` : "");
}

function showToast(title: string, message: string, color: typeof toast.color = "info", duration = 4500) {
  toast.title = title;
  toast.message = message;
  toast.color = color;
  toast.duration = duration;
  toast.visible = false;
  nextTick(() => {
    toast.visible = true;
  });
}

function setKnowledgeNotice(title: string, message: string, color: typeof knowledgeNotice.color = "info", icon = "i-heroicons-information-circle") {
  knowledgeNotice.title = title;
  knowledgeNotice.message = message;
  knowledgeNotice.color = color;
  knowledgeNotice.icon = icon;
}

async function openResultModal() {
  await nextTick();
  resultOpen.value = true;
}

function summarizeDiagnostics(diagnostics?: Record<string, unknown>) {
  if (!diagnostics) return "";
  const job = diagnostics.job as Record<string, unknown> | undefined;
  const records = diagnostics.records as unknown[] | undefined;
  const contractGap = String(diagnostics.contract_gap || "");
  const message = String(diagnostics.message || "");
  const nextAction = String(diagnostics.next_action || "");
  const note = String(diagnostics.note || "");
  const status = String(diagnostics.status || job?.status || diagnostics.provider_mode || "");
  const traceID = String(diagnostics.trace_id || "");
  if (contractGap) return contractGap;
  if (message) return nextAction ? `${message} ${nextAction}` : message;
  if (Array.isArray(records)) return records.length ? t("knowledgeLab.recordsLoaded", { count: records.length }) : t('knowledgeLab.copy.text038');
  if (status || traceID) return t("knowledgeLab.statusTrace", { status: status || "-", trace: traceID || "-" });
  if (note) return note;
  return t('knowledgeLab.copy.text039');
}

function isConflictError(error: any) {
  const code = String(error?.data?.code || error?.data?.error?.code || "").toUpperCase();
  const status = Number(error?.status || error?.statusCode || error?.response?.status || error?.data?.status || 0);
  return status === 409 || code === "KNOWLEDGE_CONFLICT";
}

function findExistingSpace(spaceName: string, departmentCode: string) {
  const name = spaceName.trim();
  const department = departmentCode.trim();
  return spaces.value.find((space) => {
    const existingName = String(space.spaceName || space.name || "").trim();
    const existingDepartment = String(space.departmentCode || space.department || "").trim();
    return existingName === name && (!department || existingDepartment === department);
  });
}

async function loadProvider() {
  providerLoading.value = true;
  providerError.value = "";
  try {
    providerState.value = unwrap<ProviderResponse>(await get("/admin/runtime/knowledge-lab/provider"));
  } catch (error: any) {
    providerError.value = errorMessage(error);
  } finally {
    providerLoading.value = false;
  }
}

async function loadKnowledgeCatalog() {
  catalogError.value = "";
  knowledgeCatalog.value = {};
  if (!providerState.value.capabilities?.operations?.includes("catalog")) {
    catalogError.value = t("knowledgeLab.catalogGap");
    return;
  }
  try {
    knowledgeCatalog.value = unwrap<KnowledgeCatalog>(await get("/admin/runtime/knowledge-lab/catalog"));
    if (!knowledgeCatalog.value.scenes?.some((scene) => scene.key === createForm.sceneKey)) {
      createForm.sceneKey = knowledgeCatalog.value.scenes?.[0]?.key || "product_specs";
    }
    if (!knowledgeCatalog.value.strategy_packages?.some((pkg) => pkg.key === createForm.strategyPackageKey)) {
      createForm.strategyPackageKey = knowledgeScenesByKey.value[createForm.sceneKey]?.default_strategy_package || knowledgeCatalog.value.strategy_packages?.[0]?.key || "H_fusion";
    }
    setCreateStrategyPackage(createForm.strategyPackageKey, createForm.sceneKey);
  } catch (error: any) {
    catalogError.value = errorMessage(error);
  }
}

async function loadSpaces() {
  spacesError.value = "";
  spaces.value = [];
  try {
    const payload = unwrap<{ spaces?: SpaceRow[]; contract_gaps?: string[] }>(await get("/admin/runtime/knowledge-lab/spaces"));
    contractGaps.value = payload.contract_gaps || [];
    spaces.value = (payload.spaces || []).map(normalizeSpace);
    const requestedSpaceID = typeof route.query.space_uuid === "string" ? route.query.space_uuid : "";
    if (requestedSpaceID && spaces.value.some((space) => space.id === requestedSpaceID)) {
      selectedSpaceId.value = requestedSpaceID;
    } else if (spaces.value.length && !spaces.value.some((space) => space.id === selectedSpaceId.value)) {
      selectedSpaceId.value = spaces.value[0]?.id || "";
    }
    if (!spaces.value.length) {
      selectedSpaceId.value = "";
    }
  } catch (error: any) {
    spacesError.value = errorMessage(error);
    selectedSpaceId.value = "";
  }
}

async function refreshKnowledgeState() {
  providerLoading.value = true;
  providerError.value = "";
  try {
    await loadProvider();
    if (providerError.value) { spaces.value = []; selectedSpaceId.value = ""; return; }
    await Promise.all([loadKnowledgeCatalog(), loadSpaces()]);
  } finally {
    providerLoading.value = false;
  }
}

async function runSearch() {
  searchLoading.value = true;
  searchError.value = "";
  result.value = null;
  ingestionJobs.value = [];
  policyPayload.value = null;
  resultKind.value = "search";
  try {
    if (!activeSpaceId.value) {
      searchError.value = t("knowledgeLab.selectSpace");
      await openResultModal();
      return;
    }
    const payload: Record<string, any> = {
      query: form.query,
      space_id: activeSpaceId.value,
      visibility: form.space_type,
      limit: form.limit,
    };
    result.value = unwrap<SearchResult>(await post("/admin/runtime/knowledge-lab/search", payload));
    await openResultModal();
  } catch (error: any) {
    searchError.value = errorMessage(error);
    await openResultModal();
  } finally {
    searchLoading.value = false;
  }
}

function openIngestionModal(space: SpaceRow) {
  const spaceId = String(space.id || space.spaceId || "").trim();
  selectSpace(spaceId);
  ingestionError.value = "";
  ingestionForm.spaceId = spaceId;
  ingestionForm.format = "markdown";
  ingestionForm.sourceUri = "";
  ingestionForm.ingestionProfile = space.ingestionProfileKey || space.ragProfileKey || "p1_general";
  ingestionForm.requestedBy = "plugin-knowledge-lab";
  ingestionSourceMethod.value = "upload";
  selectedIngestionFile.value = null;
  ingestionOpen.value = true;
}

function onIngestionFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0] || null;
  selectedIngestionFile.value = file;
  if (!file) return;
  ingestionForm.format = inferIngestionFormat(file.name);
  ingestionForm.sourceUri = "";
}

async function submitIngestion() {
  const spaceId = String(ingestionForm.spaceId || "").trim();
  if (!spaceId) {
    ingestionError.value = t('knowledgeLab.copy.text040');
    return;
  }
  if (ingestionSourceMethod.value === "upload" && !selectedIngestionFile.value) {
    ingestionError.value = t('knowledgeLab.copy.text041');
    return;
  }
  if (ingestionSourceMethod.value === "url" && !ingestionForm.sourceUri.trim()) {
    ingestionError.value = t('knowledgeLab.copy.text042');
    return;
  }
  ingestionLoading.value = true;
  ingestionError.value = "";
  try {
    let sourceUri = ingestionForm.sourceUri.trim();
    if (ingestionSourceMethod.value === "upload" && selectedIngestionFile.value) {
      const formData = new FormData();
      formData.set("file", selectedIngestionFile.value);
      formData.set("spaceId", spaceId);
      formData.set("format", ingestionForm.format);
      formData.set("requestedBy", ingestionForm.requestedBy.trim() || "plugin-knowledge-lab");
      const uploaded = unwrap<Record<string, any>>(await apiPost("/admin/runtime/knowledge-lab/media/upload", formData));
      sourceUri = String(uploaded.sourceUri || "").trim();
      if (!sourceUri) {
        throw new Error(t('knowledgeLab.copy.text043'));
      }
      if (uploaded.format) {
        ingestionForm.format = String(uploaded.format);
      }
    }
    const payload = unwrap<Record<string, unknown>>(await post(`/admin/runtime/knowledge-lab/spaces/${encodeURIComponent(spaceId)}/ingest`, {
      format: ingestionForm.format,
      sourceUri,
      ingestionProfile: ingestionForm.ingestionProfile.trim() || "p1_general",
      requestedBy: ingestionForm.requestedBy.trim() || "plugin-knowledge-lab",
    }));
    diagnosticsTitle.value = t('knowledgeLab.copy.text044');
    ingestionJobs.value = [];
    policyPayload.value = null;
    resultKind.value = "diagnostics";
    result.value = diagnosticsResult(payload);
    setKnowledgeNotice(t('knowledgeLab.copy.text044'), t('knowledgeLab.copy.text045'), "success", "i-heroicons-check-circle");
    showToast(t('knowledgeLab.copy.text044'), t('knowledgeLab.copy.text046'), "success");
    ingestionOpen.value = false;
    await loadSpaces();
    await openResultModal();
    return true;
  } catch (error: any) {
    ingestionError.value = errorMessage(error);
    showToast(t('knowledgeLab.copy.text047'), ingestionError.value, "error", 7000);
    return false;
  } finally {
    ingestionLoading.value = false;
  }
}

function inferIngestionFormat(filename: string) {
  const ext = filename.split(".").pop()?.toLowerCase() || "";
  if (ext === "md" || ext === "markdown") return "markdown";
  if (ext === "txt") return "text";
  if (ext === "pdf") return "pdf";
  if (ext === "html" || ext === "htm") return "html";
  if (ext === "csv") return "csv";
  return ingestionForm.format || "markdown";
}

function selectSpace(spaceId: string) {
  if (spaces.value.some((space) => space.id === spaceId)) {
    selectedSpaceId.value = spaceId;
    return;
  }
  selectedSpaceId.value = "";

}

async function focusPlayground(spaceId?: string) {
  if (spaceId) {
    selectSpace(spaceId);
  }
  await nextTick();
  playgroundFormRef.value?.scrollIntoView({ behavior: "smooth", block: "start" });
}

async function showIngestion(spaceId: string) {
  selectSpace(spaceId);
  searchError.value = "";
  result.value = null;
  policyPayload.value = null;
  resultKind.value = "ingestions";
  try {
    const payload = unwrap<Record<string, unknown>>(await get(`/admin/runtime/knowledge-lab/spaces/${encodeURIComponent(spaceId)}/ingestions?limit=20`));
    diagnosticsTitle.value = t('knowledgeLab.copy.text030');
    ingestionJobs.value = normalizeIngestionJobs(payload);
    ingestionPage.value = 1;
    result.value = diagnosticsResult(payload);
    setKnowledgeNotice(t('knowledgeLab.copy.text048'), summarizeDiagnostics(payload) || t('knowledgeLab.copy.text049'), "info");
    await openResultModal();
  } catch (error: any) {
    searchError.value = errorMessage(error);
    showToast(t('knowledgeLab.copy.text033'), searchError.value, "error", 7000);
    await openResultModal();
  }
}

function showDataSource(spaceId: string) {
  selectSpace(spaceId);
  ingestionJobs.value = [];
  policyPayload.value = null;
  resultKind.value = "diagnostics";
  diagnosticsTitle.value = t('knowledgeLab.copy.text050');
  const payload = {
    space_id: spaceId,
    provider: providerState.value.provider || "delegated",
    provider_mode: providerState.value.mode || "delegated",
    contract_gap: t('knowledgeLab.copy.text051'),
  };
  result.value = diagnosticsResult(payload);
  setKnowledgeNotice(t('knowledgeLab.copy.text052'), payload.contract_gap, "warning", "i-heroicons-exclamation-triangle");
  openResultModal();
}

async function showPolicy(spaceId: string) {
  selectSpace(spaceId);
  searchError.value = "";
  result.value = null;
  ingestionJobs.value = [];
  resultKind.value = "policy";
  try {
    const payload = unwrap<Record<string, unknown>>(await get(`/admin/runtime/knowledge-lab/spaces/${encodeURIComponent(spaceId)}/policy`));
    diagnosticsTitle.value = t('knowledgeLab.copy.text031');
    policyPayload.value = payload;
    result.value = diagnosticsResult(payload);
    setKnowledgeNotice(t('knowledgeLab.copy.text053'), summarizeDiagnostics(payload) || t('knowledgeLab.copy.text054'), "info");
    await openResultModal();
  } catch (error: any) {
    searchError.value = errorMessage(error);
    showToast(t('knowledgeLab.copy.text034'), searchError.value, "error", 7000);
    await openResultModal();
  }
}

function diagnosticsResult(diagnostics: Record<string, unknown>): SearchResult {
  return {
    provider: providerState.value.provider || "delegated",
    total: 0,
    chunks: [],
    citations: [],
    diagnostics,
  };
}

function normalizeIngestionJobs(payload: Record<string, unknown>): IngestionJobRecord[] {
  const records = Array.isArray(payload.records) ? payload.records : Array.isArray(payload.items) ? payload.items : [];
  return records.map((record: any) => ({
    id: String(record?.id || record?.jobId || record?.job_id || ""),
    jobId: String(record?.jobId || record?.job_id || record?.id || ""),
    status: String(record?.status || ""),
    retryCount: numberOrUndefined(record?.retryCount ?? record?.retry_count),
    errorCode: String(record?.errorCode || record?.error_code || ""),
    reason: String(record?.reason || record?.blockedReason || record?.blocked_reason || ""),
    chunkTotal: numberOrUndefined(record?.chunkTotal ?? record?.chunk_total),
    chunkCoveragePct: numberOrUndefined(record?.chunkCoveragePct ?? record?.chunk_coverage_pct),
    embeddingSuccessPct: numberOrUndefined(record?.embeddingSuccessPct ?? record?.embedding_success_pct),
    maskingCoveragePct: numberOrUndefined(record?.maskingCoveragePct ?? record?.masking_coverage_pct),
  }));
}

function normalizePolicyStrategies(payload?: Record<string, any> | null): Record<string, any>[] {
  if (!payload) return [];
  const candidates = [
    payload.strategies,
    payload.items,
    payload.records,
    payload.data,
    payload.fusion_strategy,
    payload.fusion_strategy?.data,
    payload.fusion_strategy?.items,
    payload.fusion_strategy?.records,
    payload.fusion_strategy?.payload?.data,
    payload.fusion_strategy?.payload?.items,
    payload.fusion_strategy?.result?.data,
    payload.fusion_strategy?.result?.items,
  ];
  for (const candidate of candidates) {
    if (Array.isArray(candidate)) {
      return candidate.map((item) => item && typeof item === "object" ? item as Record<string, any> : { name: String(item) });
    }
  }
  return [];
}

function numberOrUndefined(value: unknown) {
  const num = Number(value);
  return Number.isFinite(num) ? num : undefined;
}

function formatPercent(value?: number) {
  if (value === undefined || value === null || Number.isNaN(Number(value))) return "-";
  return `${Number(value).toFixed(1).replace(/\.0$/, "")}%`;
}

function ingestionStatusColor(status?: string) {
  const normalized = String(status || "").toLowerCase();
  if (["completed", "success", "succeeded"].includes(normalized)) return "success";
  if (["failed", "blocked", "error"].includes(normalized)) return "error";
  if (["running", "processing", "queued", "pending"].includes(normalized)) return "warning";
  return "neutral";
}

function normalizeSpace(space: SpaceRow): SpaceRow {
  const status = space.status || "pending";
  const id = space.id || space.spaceId || "";
  const name = space.name || space.spaceName || t("knowledgeLab.unnamedSpace");
  return {
    ...space,
    id,
    name,
    spaceId: space.spaceId || id,
    spaceName: space.spaceName || name,
    department: space.department || space.departmentCode || "-",
    departmentCode: space.departmentCode || space.department || "-",
    typeLabel: space.typeLabel || space.type_label || space.type,
    statusLabel: statusLabel(status),
    statusColor: space.statusColor || space.status_color || statusColor(status),
    icon: space.icon || "i-heroicons-circle-stack",
  };
}

function statusColor(status: string): "success" | "warning" | "neutral" | "error" | "info" {
  if (status === "active" || status === "ready") return "success";
  if (status === "pending" || status === "pending_iam") return "warning";
  if (status === "unavailable") return "error";
  return "neutral";
}

function statusLabel(status: string) {
  if (status === "active") return t('knowledgeLab.copy.text016');
  if (status === "pending_iam") return t('knowledgeLab.copy.text017');
  if (status === "retired") return t('knowledgeLab.copy.text018');
  if (status === "ready") return t('knowledgeLab.copy.text055');
  if (status === "pending") return t('knowledgeLab.copy.text019');
  if (status === "unavailable") return t('knowledgeLab.copy.text020');
  return status;
}

function providerModeLabel(mode?: string) {
  if (mode === "delegated") return t("knowledgeLab.proxyMode");
  if (mode === "local") return t('knowledgeLab.copy.text021');
  if (mode === "mock") return t('knowledgeLab.copy.text023');
  return mode || "-";
}

function healthLabel(health?: string) {
  if (health === "ready") return t('knowledgeLab.copy.text056');
  if (health === "degraded") return t('knowledgeLab.copy.text057');
  if (health === "unavailable") return t('knowledgeLab.copy.text020');
  return health || "-";
}

function spaceTypeLabel(space: SpaceRow) {
  return space.typeLabel || space.type_label || space.type;
}

function compactId(value: string) {
  if (!value) return "-";
  return value.length > 12 ? `${value.slice(0, 8)}...` : value;
}

function quotaValue(space: SpaceRow, key: "cpuCores" | "storageGb" | "ingestionConcurrency") {
  const value = space.quotas?.[key];
  if (value === undefined || value === null || value === "") return "-";
  return String(value);
}

function formatScore(score?: number) {
  if (typeof score !== "number") return "-";
  return score.toFixed(2);
}

onMounted(() => {
  refreshKnowledgeState();
});
</script>
