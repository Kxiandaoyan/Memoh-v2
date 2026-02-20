<template>
  <div class="max-w-4xl mx-auto space-y-5">
    <!-- Built-in tools -->
    <div class="rounded-lg border bg-card">
      <button
        class="flex w-full items-center justify-between px-4 py-3 text-left"
        @click="builtinExpanded = !builtinExpanded"
      >
        <div class="space-y-0.5">
          <h3 class="text-sm font-semibold">
            {{ $t('mcp.builtinTitle') }}
          </h3>
          <p class="text-xs text-muted-foreground">
            {{ $t('mcp.builtinDescription', { count: builtinTools.length }) }}
          </p>
        </div>
        <svg
          class="size-4 shrink-0 text-muted-foreground transition-transform duration-200"
          :class="{ 'rotate-180': builtinExpanded }"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </button>
      <div
        v-if="builtinExpanded"
        class="border-t px-4 py-3"
      >
        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="tool in builtinTools"
            :key="tool.name"
            class="flex items-start gap-2 rounded-md border px-3 py-2"
          >
            <span class="mt-0.5 text-xs font-mono font-medium text-primary whitespace-nowrap">{{ tool.name }}</span>
            <span class="text-xs text-muted-foreground leading-relaxed">{{ tool.desc }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- External MCP connections -->
    <div class="flex items-start justify-between gap-3">
      <div class="space-y-1 min-w-0">
        <h3 class="text-lg font-semibold">
          {{ $t('mcp.externalTitle') }}
        </h3>
        <p class="text-sm text-muted-foreground">
          {{ $t('mcp.externalDescription') }}
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2 shrink-0 justify-end">
        <template v-if="selectedIds.length === 0">
          <Button
            variant="outline"
            size="sm"
            :disabled="loading"
            @click="loadList"
          >
            <Spinner
              v-if="loading"
              class="mr-1.5"
            />
            {{ $t('common.refresh') }}
          </Button>
          <Button
            size="sm"
            @click="openCreateDialog"
          >
            {{ $t('common.add') }}
          </Button>
        </template>
        <template v-else>
          <span class="text-sm text-muted-foreground mr-1">
            {{ $t('common.batchSelected', { count: selectedIds.length }) }}
          </span>
          <Button
            variant="ghost"
            size="sm"
            @click="clearSelection"
          >
            {{ $t('common.cancelSelection') }}
          </Button>
          <Button
            variant="outline"
            size="sm"
            @click="handleBatchExport"
          >
            {{ $t('common.export') }}
          </Button>
          <ConfirmPopover
            :message="$t('common.batchDeleteConfirm', { count: selectedIds.length })"
            @confirm="handleBatchDelete"
          >
            <template #trigger>
              <Button
                variant="destructive"
                size="sm"
              >
                {{ $t('common.delete') }}
              </Button>
            </template>
          </ConfirmPopover>
        </template>
      </div>
    </div>

    <!-- Loading -->
    <div
      v-if="loading && items.length === 0"
      class="flex items-center gap-2 text-sm text-muted-foreground"
    >
      <Spinner />
      <span>{{ $t('common.loading') }}</span>
    </div>

    <!-- Empty -->
    <div
      v-else-if="items.length === 0"
      class="rounded-md border p-4"
    >
      <p class="text-sm text-muted-foreground">
        {{ $t('mcp.empty') }}
      </p>
    </div>

    <!-- Table -->
    <DataTable
      v-else
      :columns="columns"
      :data="items"
    />

    <!-- Marketplace -->
    <div class="rounded-lg border bg-card">
      <button
        class="flex w-full items-center justify-between px-4 py-3 text-left"
        @click="marketplaceExpanded = !marketplaceExpanded"
      >
        <div class="space-y-0.5">
          <h3 class="text-sm font-semibold">
            {{ $t('mcp.marketplace.title') }}
          </h3>
          <p class="text-xs text-muted-foreground">
            {{ $t('mcp.marketplace.poweredBy') }}
          </p>
        </div>
        <svg
          class="size-4 shrink-0 text-muted-foreground transition-transform duration-200"
          :class="{ 'rotate-180': marketplaceExpanded }"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </button>
      <div
        v-if="marketplaceExpanded"
        class="border-t"
      >
        <Tabs
          v-model="marketplaceTab"
          class="w-full"
        >
          <TabsList class="w-full rounded-none border-b bg-transparent px-4 pt-2">
            <TabsTrigger value="smithery">
              Smithery
            </TabsTrigger>
            <TabsTrigger value="modelscope">
              ModelScope
            </TabsTrigger>
          </TabsList>

          <!-- Smithery tab -->
          <TabsContent
            value="smithery"
            class="px-4 py-3 space-y-3"
          >
            <!-- API key hint -->
            <div
              v-if="mpSearchError"
              class="rounded-md border border-yellow-500/30 bg-yellow-500/5 px-3 py-2 text-xs text-yellow-700 dark:text-yellow-400"
            >
              {{ $t('mcp.marketplace.apiKeyHint') }}
            </div>

            <div class="flex gap-2">
              <Input
                v-model="mpQuery"
                :placeholder="$t('mcp.marketplace.searchPlaceholder')"
                class="flex-1"
                @input="mpDebouncedSearch"
                @keydown.enter="mpPage = 1; mpSearch()"
              />
              <Button
                size="sm"
                :disabled="mpSearching"
                @click="mpPage = 1; mpSearch()"
              >
                <Spinner
                  v-if="mpSearching"
                  class="mr-1.5"
                />
                {{ $t('common.search') }}
              </Button>
            </div>

            <!-- No results or hint -->
            <div
              v-if="!mpSearching && mpServers.length === 0"
              class="py-6 text-center text-sm text-muted-foreground"
            >
              {{ mpQuery ? $t('mcp.marketplace.noResults') : $t('mcp.marketplace.searchHint') }}
            </div>

            <!-- Results grid -->
            <div
              v-else
              class="space-y-2"
            >
              <div class="grid gap-2 sm:grid-cols-2">
                <template
                  v-for="server in mpServers"
                  :key="server.qualifiedName"
                >
                  <div
                    class="rounded-md border p-3 cursor-pointer transition-colors hover:bg-muted/50"
                    :class="{ 'ring-1 ring-primary': mpExpandedName === server.qualifiedName }"
                    @click="mpToggleDetail(server)"
                  >
                    <div class="flex items-start gap-2.5">
                      <img
                        v-if="server.iconUrl"
                        :src="server.iconUrl"
                        :alt="server.displayName"
                        class="size-8 rounded object-cover shrink-0 mt-0.5"
                        @error="($event.target as HTMLImageElement).style.display = 'none'"
                      >
                      <div
                        v-else
                        class="size-8 rounded bg-muted flex items-center justify-center shrink-0 mt-0.5"
                      >
                        <svg
                          class="size-4 text-muted-foreground"
                          xmlns="http://www.w3.org/2000/svg"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          stroke-width="2"
                          stroke-linecap="round"
                          stroke-linejoin="round"
                        >
                          <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
                        </svg>
                      </div>
                      <div class="min-w-0 flex-1">
                        <div class="flex items-center gap-1.5">
                          <span class="text-sm font-medium truncate">{{ server.displayName }}</span>
                          <Badge
                            v-if="server.verified"
                            variant="default"
                            class="text-[10px] px-1 py-0 shrink-0"
                          >
                            {{ $t('mcp.marketplace.verified') }}
                          </Badge>
                        </div>
                        <p class="text-xs text-muted-foreground line-clamp-2 mt-0.5">
                          {{ server.description }}
                        </p>
                        <div class="flex items-center gap-2 mt-1.5 text-[11px] text-muted-foreground">
                          <Badge
                            variant="outline"
                            class="text-[10px] px-1 py-0"
                          >
                            {{ server.remote ? $t('mcp.marketplace.remote') : $t('mcp.marketplace.local') }}
                          </Badge>
                          <span v-if="server.useCount">{{ server.useCount.toLocaleString() }} {{ $t('mcp.marketplace.uses') }}</span>
                        </div>
                      </div>
                      <Badge
                        v-if="mpIsInstalled(server.qualifiedName)"
                        variant="secondary"
                        class="shrink-0 text-[10px]"
                      >
                        {{ $t('mcp.marketplace.installed') }}
                      </Badge>
                    </div>
                  </div>

                  <!-- Expanded detail panel -->
                  <div
                    v-if="mpExpandedName === server.qualifiedName"
                    class="sm:col-span-2 rounded-md border bg-muted/30 p-4 space-y-3"
                  >
                    <div
                      v-if="mpDetailLoading"
                      class="flex items-center gap-2 text-sm text-muted-foreground"
                    >
                      <Spinner />
                      <span>{{ $t('mcp.marketplace.loadingDetail') }}</span>
                    </div>
                    <template v-else-if="mpDetail">
                      <div class="flex items-start justify-between gap-3">
                        <div>
                          <h4 class="text-sm font-semibold">
                            {{ mpDetail.displayName }}
                          </h4>
                          <p class="text-xs text-muted-foreground mt-0.5">
                            {{ mpDetail.qualifiedName }}
                          </p>
                        </div>
                        <div class="flex items-center gap-2 shrink-0">
                          <a
                            :href="`https://smithery.ai/server/${mpDetail.qualifiedName}`"
                            target="_blank"
                            rel="noopener"
                            class="text-xs text-primary hover:underline"
                          >
                            {{ $t('mcp.marketplace.viewOnSmithery') }}
                          </a>
                          <template v-if="mpHasHttpConnection(mpDetail)">
                            <Button
                              size="sm"
                              :disabled="mpInstallingName === mpDetail.qualifiedName || mpIsInstalled(mpDetail.qualifiedName)"
                              @click.stop="mpInstall(mpDetail!)"
                            >
                              <Spinner
                                v-if="mpInstallingName === mpDetail.qualifiedName"
                                class="mr-1.5"
                              />
                              {{ mpIsInstalled(mpDetail.qualifiedName) ? $t('mcp.marketplace.installed') : $t('mcp.marketplace.install') }}
                            </Button>
                          </template>
                          <template v-else-if="mpDetail.connections?.some((c: any) => c.type === 'stdio')">
                            <Button
                              size="sm"
                              variant="outline"
                              :disabled="mpIsInstalled(mpDetail.qualifiedName)"
                              @click.stop="mpPrefillStdioForm(mpDetail!)"
                            >
                              {{ mpIsInstalled(mpDetail.qualifiedName) ? $t('mcp.marketplace.installed') : $t('mcp.marketplace.manualSetup') }}
                            </Button>
                          </template>
                          <span
                            v-else
                            class="text-xs text-muted-foreground italic"
                          >
                            {{ $t('mcp.marketplace.noConnectionInfo') }}
                          </span>
                        </div>
                      </div>
                      <p class="text-xs text-muted-foreground">
                        {{ mpDetail.description }}
                      </p>
                      <!-- Tools list -->
                      <div v-if="mpDetail.tools && mpDetail.tools.length > 0">
                        <p class="text-xs font-medium mb-1.5">
                          {{ mpDetail.tools.length }} {{ $t('mcp.marketplace.tools') }}
                        </p>
                        <div class="grid gap-1.5 sm:grid-cols-2">
                          <div
                            v-for="tool in mpDetail.tools"
                            :key="tool.name"
                            class="flex items-start gap-2 rounded border bg-background px-2.5 py-1.5"
                          >
                            <span class="text-[11px] font-mono font-medium text-primary whitespace-nowrap">{{ tool.name }}</span>
                            <span
                              v-if="tool.description"
                              class="text-[11px] text-muted-foreground line-clamp-1"
                            >{{ tool.description }}</span>
                          </div>
                        </div>
                      </div>
                    </template>
                  </div>
                </template>
              </div>

              <!-- Pagination -->
              <div
                v-if="mpTotalPages > 1"
                class="flex items-center justify-between pt-1"
              >
                <span class="text-xs text-muted-foreground">
                  {{ mpTotalCount.toLocaleString() }} servers
                </span>
                <div class="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    :disabled="mpPage <= 1"
                    @click="mpPrevPage"
                  >
                    &larr;
                  </Button>
                  <span class="text-xs text-muted-foreground">
                    {{ mpPage }} / {{ mpTotalPages }}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    :disabled="mpPage >= mpTotalPages"
                    @click="mpNextPage"
                  >
                    &rarr;
                  </Button>
                </div>
              </div>
            </div>
          </TabsContent>

          <!-- ModelScope tab -->
          <TabsContent
            value="modelscope"
            class="px-4 py-3"
          >
            <div class="rounded-md border bg-muted/30 p-4 space-y-3">
              <div class="flex items-center gap-2.5">
                <div class="size-8 rounded bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center shrink-0">
                  <span class="text-white text-xs font-bold">MS</span>
                </div>
                <div>
                  <h4 class="text-sm font-semibold">
                    {{ $t('mcp.marketplace.modelscope.title') }}
                  </h4>
                </div>
              </div>
              <p class="text-sm text-muted-foreground">
                {{ $t('mcp.marketplace.modelscope.description') }}
              </p>
              <p class="text-xs text-muted-foreground">
                {{ $t('mcp.marketplace.modelscope.hint') }}
              </p>
              <a
                href="https://www.modelscope.cn/mcp"
                target="_blank"
                rel="noopener"
              >
                <Button
                  variant="outline"
                  size="sm"
                  class="mt-1"
                >
                  {{ $t('mcp.marketplace.modelscope.visit') }}
                  <svg
                    class="ml-1 size-3.5"
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
                    <polyline points="15 3 21 3 21 9" />
                    <line
                      x1="10"
                      y1="14"
                      x2="21"
                      y2="3"
                    />
                  </svg>
                </Button>
              </a>
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </div>

    <!-- Add dialog: tabs (single | import). Edit dialog: two columns (form | json) with sync -->
    <Dialog v-model:open="formDialogOpen">
      <DialogContent :class="editingItem ? 'sm:max-w-4xl max-h-[90vh] flex flex-col w-[calc(100vw-2rem)] max-w-[calc(100vw-2rem)] sm:w-auto sm:max-w-full' : 'sm:max-w-[28rem] w-[calc(100vw-2rem)] max-w-[calc(100vw-2rem)] sm:w-auto'">
        <DialogHeader>
          <DialogTitle>{{ editingItem ? $t('common.edit') : $t('common.add') }} MCP Server</DialogTitle>
        </DialogHeader>

        <!-- Edit: two columns on desktop, stacked on mobile -->
        <template v-if="editingItem">
          <div class="mt-3 flex flex-col md:grid md:grid-cols-2 gap-4 flex-1 min-h-0 overflow-y-auto">
            <form
              class="flex flex-col gap-3 min-h-0 rounded-lg border border-border bg-card p-3 md:bg-transparent md:border-0 md:p-0 md:rounded-none md:overflow-y-auto md:pr-2"
              @submit.prevent="handleSubmit"
            >
              <div class="space-y-1.5">
                <Label>{{ $t('common.name') }}</Label>
                <Input
                  v-model="formData.name"
                  :placeholder="$t('common.namePlaceholder')"
                  @update:model-value="syncFormToEditJson"
                />
              </div>
              <Tabs
                v-model="connectionMode"
                class="w-full"
              >
                <TabsList class="w-full">
                  <TabsTrigger value="stdio">
                    {{ $t('mcp.types.stdio') }}
                  </TabsTrigger>
                  <TabsTrigger value="remote">
                    {{ $t('mcp.types.remote') }}
                  </TabsTrigger>
                </TabsList>
                <TabsContent
                  value="stdio"
                  class="mt-3 flex flex-col gap-3"
                >
                  <div class="space-y-1.5">
                    <Label>{{ $t('mcp.command') }}</Label>
                    <p class="text-xs text-muted-foreground">{{ $t('mcp.commandHint') }}</p>
                    <Input
                      v-model="formData.command"
                      :placeholder="$t('mcp.commandPlaceholder')"
                      @update:model-value="syncFormToEditJson"
                    />
                  </div>
                  <div class="space-y-1.5">
                    <Label>{{ $t('mcp.arguments') }}</Label>
                    <p class="text-xs text-muted-foreground">{{ $t('mcp.argumentsHint') }}</p>
                    <TagsInput
                      v-model="argsTags"
                      :add-on-blur="true"
                      :duplicate="true"
                      @update:model-value="syncFormToEditJson"
                    >
                      <TagsInputItem
                        v-for="item in argsTags"
                        :key="item"
                        :value="item"
                      >
                        <TagsInputItemText />
                        <TagsInputItemDelete />
                      </TagsInputItem>
                      <TagsInputInput
                        :placeholder="$t('mcp.argumentsPlaceholder')"
                        class="w-full py-1"
                      />
                    </TagsInput>
                  </div>
                  <div class="space-y-1.5">
                    <Label>{{ $t('mcp.env') }}</Label>
                    <p class="text-xs text-muted-foreground">{{ $t('mcp.envHint') }}</p>
                    <TagsInput
                      :model-value="envTags.tagList.value"
                      :add-on-blur="true"
                      :convert-value="envTags.convertValue"
                      @update:model-value="(tags) => { envTags.handleUpdate(tags.map(String)); syncFormToEditJson() }"
                    >
                      <TagsInputItem
                        v-for="(value, index) in envTags.tagList.value"
                        :key="index"
                        :value="value"
                      >
                        <TagsInputItemText />
                        <TagsInputItemDelete />
                      </TagsInputItem>
                      <TagsInputInput
                        :placeholder="$t('mcp.envPlaceholder')"
                        class="w-full py-1"
                      />
                    </TagsInput>
                  </div>
                  <div class="space-y-1.5">
                    <Label>{{ $t('mcp.cwd') }}</Label>
                    <p class="text-xs text-muted-foreground">{{ $t('mcp.cwdHint') }}</p>
                    <Input
                      v-model="formData.cwd"
                      :placeholder="$t('mcp.cwdPlaceholder')"
                      @update:model-value="syncFormToEditJson"
                    />
                  </div>
                </TabsContent>
                <TabsContent
                  value="remote"
                  class="mt-3 flex flex-col gap-3"
                >
                  <div class="space-y-1.5">
                    <Label>URL</Label>
                    <p class="text-xs text-muted-foreground">{{ $t('mcp.urlHint') }}</p>
                    <Input
                      v-model="formData.url"
                      placeholder="https://example.com/mcp"
                      @update:model-value="syncFormToEditJson"
                    />
                  </div>
                  <div class="space-y-1.5">
                    <Label>Headers</Label>
                    <p class="text-xs text-muted-foreground">{{ $t('mcp.headersHint') }}</p>
                    <TagsInput
                      :model-value="headerTags.tagList.value"
                      :add-on-blur="true"
                      :convert-value="headerTags.convertValue"
                      @update:model-value="(tags) => { headerTags.handleUpdate(tags.map(String)); syncFormToEditJson() }"
                    >
                      <TagsInputItem
                        v-for="(value, index) in headerTags.tagList.value"
                        :key="index"
                        :value="value"
                      >
                        <TagsInputItemText />
                        <TagsInputItemDelete />
                      </TagsInputItem>
                      <TagsInputInput
                        placeholder="Key:Value"
                        class="w-full py-1"
                      />
                    </TagsInput>
                  </div>
                  <div class="space-y-1.5">
                    <Label>Transport</Label>
                    <p class="text-xs text-muted-foreground">{{ $t('mcp.transportHint') }}</p>
                    <Select
                      v-model="formData.transport"
                      @update:model-value="syncFormToEditJson"
                    >
                      <SelectTrigger class="w-full">
                        <SelectValue placeholder="http" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          <SelectItem value="http">
                            HTTP (Streamable)
                          </SelectItem>
                          <SelectItem value="sse">
                            SSE
                          </SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  </div>
                </TabsContent>
              </Tabs>
            </form>
            <div class="flex flex-col min-h-0 rounded-lg border border-border bg-card p-3 md:bg-transparent md:border-0 md:p-0 md:rounded-none">
              <Label class="text-sm mb-1">JSON</Label>
              <Textarea
                v-model="editJson"
                class="font-mono text-xs flex-1 min-h-[180px] md:min-h-[200px]"
                @update:model-value="syncEditJsonToForm"
              />
            </div>
          </div>
          <DialogFooter class="mt-4 flex-shrink-0 flex-row flex-wrap items-center gap-2 sm:justify-between">
            <div class="flex items-center gap-2">
              <Label class="text-sm font-normal">{{ $t('mcp.active') }}</Label>
              <Switch
                :model-value="formData.active"
                @update:model-value="(val) => (formData.active = !!val)"
              />
            </div>
            <div class="flex gap-2">
              <DialogClose as-child>
                <Button variant="outline">
                  {{ $t('common.cancel') }}
                </Button>
              </DialogClose>
              <Button
                :disabled="submitting || !formData.name.trim() || (connectionMode === 'stdio' ? !formData.command.trim() : !formData.url.trim())"
                @click="handleSubmit"
              >
                <Spinner
                  v-if="submitting"
                  class="mr-1.5"
                />
                {{ $t('common.confirm') }}
              </Button>
            </div>
          </DialogFooter>
        </template>

        <!-- Add: tabs single | import -->
        <template v-else>
          <Tabs
            v-model="addDialogTab"
            class="mt-4 w-full"
          >
            <TabsList class="w-full">
              <TabsTrigger value="single">
                {{ $t('common.tabAddSingle') }}
              </TabsTrigger>
              <TabsTrigger value="import">
                {{ $t('common.tabImportJson') }}
              </TabsTrigger>
            </TabsList>

            <TabsContent
              value="single"
              class="mt-3"
            >
              <form
                class="flex flex-col gap-3"
                @submit.prevent="handleSubmit"
              >
                <div class="space-y-1.5">
                  <Label>{{ $t('common.name') }}</Label>
                  <Input
                    v-model="formData.name"
                    :placeholder="$t('common.namePlaceholder')"
                  />
                </div>
                <Tabs
                  v-model="connectionMode"
                  class="w-full"
                >
                  <TabsList class="w-full">
                    <TabsTrigger value="stdio">
                      {{ $t('mcp.types.stdio') }}
                    </TabsTrigger>
                    <TabsTrigger value="remote">
                      {{ $t('mcp.types.remote') }}
                    </TabsTrigger>
                  </TabsList>
                  <TabsContent
                    value="stdio"
                    class="mt-3 flex flex-col gap-3"
                  >
                    <div class="space-y-1.5">
                      <Label>{{ $t('mcp.command') }}</Label>
                      <p class="text-xs text-muted-foreground">{{ $t('mcp.commandHint') }}</p>
                      <Input
                        v-model="formData.command"
                        :placeholder="$t('mcp.commandPlaceholder')"
                      />
                    </div>
                    <div class="space-y-1.5">
                      <Label>{{ $t('mcp.arguments') }}</Label>
                      <p class="text-xs text-muted-foreground">{{ $t('mcp.argumentsHint') }}</p>
                      <TagsInput
                        v-model="argsTags"
                        :add-on-blur="true"
                        :duplicate="true"
                      >
                        <TagsInputItem
                          v-for="item in argsTags"
                          :key="item"
                          :value="item"
                        >
                          <TagsInputItemText />
                          <TagsInputItemDelete />
                        </TagsInputItem>
                        <TagsInputInput
                          :placeholder="$t('mcp.argumentsPlaceholder')"
                          class="w-full py-1"
                        />
                      </TagsInput>
                    </div>
                    <div class="space-y-1.5">
                      <Label>{{ $t('mcp.env') }}</Label>
                      <p class="text-xs text-muted-foreground">{{ $t('mcp.envHint') }}</p>
                      <TagsInput
                        :model-value="envTags.tagList.value"
                        :add-on-blur="true"
                        :convert-value="envTags.convertValue"
                        @update:model-value="(tags) => envTags.handleUpdate(tags.map(String))"
                      >
                        <TagsInputItem
                          v-for="(value, index) in envTags.tagList.value"
                          :key="index"
                          :value="value"
                        >
                          <TagsInputItemText />
                          <TagsInputItemDelete />
                        </TagsInputItem>
                        <TagsInputInput
                          :placeholder="$t('mcp.envPlaceholder')"
                          class="w-full py-1"
                        />
                      </TagsInput>
                    </div>
                    <div class="space-y-1.5">
                      <Label>{{ $t('mcp.cwd') }}</Label>
                      <p class="text-xs text-muted-foreground">{{ $t('mcp.cwdHint') }}</p>
                      <Input
                        v-model="formData.cwd"
                        :placeholder="$t('mcp.cwdPlaceholder')"
                      />
                    </div>
                  </TabsContent>
                  <TabsContent
                    value="remote"
                    class="mt-3 flex flex-col gap-3"
                  >
                    <div class="space-y-1.5">
                      <Label>URL</Label>
                      <p class="text-xs text-muted-foreground">{{ $t('mcp.urlHint') }}</p>
                      <Input
                        v-model="formData.url"
                        placeholder="https://example.com/mcp"
                      />
                    </div>
                    <div class="space-y-1.5">
                      <Label>Headers</Label>
                      <p class="text-xs text-muted-foreground">{{ $t('mcp.headersHint') }}</p>
                      <TagsInput
                        :model-value="headerTags.tagList.value"
                        :add-on-blur="true"
                        :convert-value="headerTags.convertValue"
                        @update:model-value="(tags) => headerTags.handleUpdate(tags.map(String))"
                      >
                        <TagsInputItem
                          v-for="(value, index) in headerTags.tagList.value"
                          :key="index"
                          :value="value"
                        >
                          <TagsInputItemText />
                          <TagsInputItemDelete />
                        </TagsInputItem>
                        <TagsInputInput
                          placeholder="Key:Value"
                          class="w-full py-1"
                        />
                      </TagsInput>
                    </div>
                    <div class="space-y-1.5">
                      <Label>Transport</Label>
                      <p class="text-xs text-muted-foreground">{{ $t('mcp.transportHint') }}</p>
                      <Select v-model="formData.transport">
                        <SelectTrigger class="w-full">
                          <SelectValue placeholder="http" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectItem value="http">
                              HTTP (Streamable)
                            </SelectItem>
                            <SelectItem value="sse">
                              SSE
                            </SelectItem>
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </div>
                  </TabsContent>
                </Tabs>
                <DialogFooter class="mt-4 flex-row flex-wrap items-center gap-2 sm:justify-between">
                  <div class="flex items-center gap-2">
                    <Label class="text-sm font-normal">{{ $t('mcp.active') }}</Label>
                    <Switch
                      :model-value="formData.active"
                      @update:model-value="(val) => (formData.active = !!val)"
                    />
                  </div>
                  <div class="flex gap-2">
                    <DialogClose as-child>
                      <Button variant="outline">
                        {{ $t('common.cancel') }}
                      </Button>
                    </DialogClose>
                    <Button
                      type="submit"
                      :disabled="submitting || !formData.name.trim() || (connectionMode === 'stdio' ? !formData.command.trim() : !formData.url.trim())"
                    >
                      <Spinner
                        v-if="submitting"
                        class="mr-1.5"
                      />
                      {{ $t('common.confirm') }}
                    </Button>
                  </div>
                </DialogFooter>
              </form>
            </TabsContent>

            <TabsContent
              value="import"
              class="mt-3 space-y-3"
            >
              <p class="text-sm text-muted-foreground">
                {{ $t('mcp.importHint') }}
              </p>
              <Textarea
                v-model="importJson"
                rows="10"
                class="font-mono text-xs"
                :placeholder="importJsonPlaceholder"
              />
              <DialogFooter class="mt-4">
                <DialogClose as-child>
                  <Button variant="outline">
                    {{ $t('common.cancel') }}
                  </Button>
                </DialogClose>
                <Button
                  :disabled="importSubmitting || !importJson.trim()"
                  @click="handleImport"
                >
                  <Spinner
                    v-if="importSubmitting"
                    class="mr-1.5"
                  />
                  {{ $t('common.import') }}
                </Button>
              </DialogFooter>
            </TabsContent>
          </Tabs>
        </template>
      </DialogContent>
    </Dialog>

    <!-- Export dialog -->
    <Dialog v-model:open="exportDialogOpen">
      <DialogContent class="sm:max-w-lg w-[calc(100vw-2rem)] max-w-[calc(100vw-2rem)] sm:w-auto">
        <DialogHeader>
          <DialogTitle>{{ $t('common.export') }} mcpServers</DialogTitle>
        </DialogHeader>
        <div class="mt-4">
          <Textarea
            :model-value="exportJson"
            rows="10"
            class="font-mono text-xs"
            readonly
          />
        </div>
        <DialogFooter class="mt-4">
          <Button
            variant="outline"
            @click="handleCopyExport"
          >
            {{ $t('common.copy') }}
          </Button>
          <DialogClose as-child>
            <Button>
              {{ $t('common.confirm') }}
            </Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { resolveErrorMessage } from '@/utils/error'
import { type ColumnDef } from '@tanstack/vue-table'
import {
  Badge,
  Button,
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Spinner,
  Switch,
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText,
  Textarea,
} from '@memoh/ui'
import DataTable from '@/components/data-table/index.vue'
import { useKeyValueTags } from '@/composables/useKeyValueTags'
import {
  getBotsByBotIdMcp,
  postBotsByBotIdMcp,
  putBotsByBotIdMcpById,
  deleteBotsByBotIdMcpById,
  postBotsByBotIdMcpOpsBatchDelete,
} from '@memoh/sdk'
import ConfirmPopover from '@/components/confirm-popover/index.vue'
import { client } from '@memoh/sdk/client'

interface SmitheryServer {
  qualifiedName: string
  displayName: string
  description: string
  iconUrl: string | null
  verified: boolean
  useCount: number
  remote: boolean | null
  isDeployed: boolean
  homepage: string
}

interface SmitheryDetail {
  qualifiedName: string
  displayName: string
  description: string
  iconUrl: string | null
  remote: boolean
  deploymentUrl: string | null
  connections: Array<{ type: string; deploymentUrl?: string; configSchema?: Record<string, unknown> }>
  tools: Array<{ name: string; description: string | null }> | null
}

interface McpItem {
  id: string
  name: string
  type: string
  config: Record<string, unknown>
  is_active: boolean
}

interface McpServerEntry {
  command?: string
  args?: string[]
  env?: Record<string, string>
  cwd?: string
  url?: string
  headers?: Record<string, string>
  transport?: string
}

const props = defineProps<{ botId: string }>()
const { t } = useI18n()

const builtinExpanded = ref(false)
const builtinTools = computed(() => [
  { name: 'read', desc: t('mcp.builtin.read') },
  { name: 'write', desc: t('mcp.builtin.write') },
  { name: 'list', desc: t('mcp.builtin.list') },
  { name: 'edit', desc: t('mcp.builtin.edit') },
  { name: 'exec', desc: t('mcp.builtin.exec') },
  { name: 'send', desc: t('mcp.builtin.send') },
  { name: 'react', desc: t('mcp.builtin.react') },
  { name: 'lookup_channel_user', desc: t('mcp.builtin.lookupChannelUser') },
  { name: 'search_memory', desc: t('mcp.builtin.searchMemory') },
  { name: 'web_search', desc: t('mcp.builtin.webSearch') },
  { name: 'list_schedule', desc: t('mcp.builtin.listSchedule') },
  { name: 'get_schedule', desc: t('mcp.builtin.getSchedule') },
  { name: 'create_schedule', desc: t('mcp.builtin.createSchedule') },
  { name: 'update_schedule', desc: t('mcp.builtin.updateSchedule') },
  { name: 'delete_schedule', desc: t('mcp.builtin.deleteSchedule') },
])

const loading = ref(false)
const items = ref<McpItem[]>([])
const formDialogOpen = ref(false)
const editingItem = ref<McpItem | null>(null)
const submitting = ref(false)
const addDialogTab = ref<'single' | 'import'>('single')
const importJsonPlaceholder = `{
  "mcpServers": {
    "hello": {
      "command": "npx",
      "args": ["-y", "mcp-hello-world"]
    }
  }
}`
const importJson = ref('')
const importSubmitting = ref(false)
const exportDialogOpen = ref(false)
const exportJson = ref('')
const selectedIds = ref<string[]>([])

const connectionMode = ref<'stdio' | 'remote'>('stdio')

const formData = ref({
  name: '',
  command: '',
  url: '',
  cwd: '',
  transport: 'http',
  active: true,
})

// Edit dialog: JSON panel synced with formData
const editJson = ref('')
let editSyncFromJson = false

watch(connectionMode, (mode) => {
  if (mode === 'stdio') {
    formData.value.url = ''
    formData.value.transport = 'http'
    headerTags.initFromObject(null)
  } else {
    formData.value.command = ''
    formData.value.cwd = ''
    argsTags.value = []
    envTags.initFromObject(null)
  }
})
const argsTags = ref<string[]>([])
const envTags = useKeyValueTags()
const headerTags = useKeyValueTags()

function configValue(config: Record<string, unknown>, key: string): string {
  const val = config?.[key]
  return typeof val === 'string' ? val : ''
}

function configArray(config: Record<string, unknown>, key: string): string[] {
  const val = config?.[key]
  if (Array.isArray(val)) return val.map(String)
  return []
}

function configMap(config: Record<string, unknown>, key: string): Record<string, string> {
  const val = config?.[key]
  if (val && typeof val === 'object' && !Array.isArray(val)) {
    const out: Record<string, string> = {}
    for (const [k, v] of Object.entries(val)) {
      out[k] = String(v)
    }
    return out
  }
  return {}
}

function toggleSelection(id: string, checked: boolean) {
  const set = new Set(selectedIds.value)
  if (checked) set.add(id)
  else set.delete(id)
  selectedIds.value = Array.from(set)
}

function toggleSelectAll(checked: boolean) {
  selectedIds.value = checked ? items.value.map((i) => i.id) : []
}

const isAllSelected = computed(() =>
  items.value.length > 0 && selectedIds.value.length === items.value.length,
)

function clearSelection() {
  selectedIds.value = []
}

function itemToExportEntry(item: McpItem): McpServerEntry {
  const cfg = item.config ?? {}
  if (item.type === 'stdio') {
    const entry: McpServerEntry = {
      command: configValue(cfg, 'command') || undefined,
      args: configArray(cfg, 'args').length ? configArray(cfg, 'args') : undefined,
      cwd: configValue(cfg, 'cwd') || undefined,
      env: Object.keys(configMap(cfg, 'env')).length ? configMap(cfg, 'env') : undefined,
    }
    return entry
  }
  const entry: McpServerEntry = {
    url: configValue(cfg, 'url') || undefined,
    headers: Object.keys(configMap(cfg, 'headers')).length ? configMap(cfg, 'headers') : undefined,
    transport: item.type === 'sse' ? 'sse' : undefined,
  }
  return entry
}

const columns = computed<ColumnDef<McpItem>[]>(() => [
  {
    id: 'select',
    header: () =>
      h('div', { class: 'flex items-center justify-center py-4' }, [
        h('input', {
          type: 'checkbox',
          class: 'size-4 cursor-pointer rounded border border-input',
          checked: isAllSelected.value,
          onChange: (e: Event) => {
            toggleSelectAll((e.target as HTMLInputElement).checked)
          },
        }),
      ]),
    cell: ({ row }) => {
      const id = row.original.id
      return h('div', { class: 'flex justify-center' }, [
        h('input', {
          type: 'checkbox',
          class: 'size-4 cursor-pointer rounded border border-input',
          checked: selectedIds.value.includes(id),
          onChange: (e: Event) => {
            toggleSelection(id, (e.target as HTMLInputElement).checked)
          },
        }),
      ])
    },
  },
  {
    accessorKey: 'name',
    header: () => h('div', { class: 'text-left py-4' }, t('common.name')),
  },
  {
    accessorKey: 'type',
    header: () => h('div', { class: 'text-left' }, t('common.type')),
    cell: ({ row }) => h(Badge, { variant: 'outline' }, () => row.original.type),
  },
  {
    id: 'target',
    header: () => h('div', { class: 'text-left' }, 'Command / URL'),
    cell: ({ row }) => {
      const cfg = row.original.config ?? {}
      const cmd = configValue(cfg, 'command')
      const url = configValue(cfg, 'url')
      const args = configArray(cfg, 'args')
      const full =
        cmd
          ? (args.length ? `${cmd} ${args.join(' ')}` : cmd)
          : (url || '-')
      return h('span', {
        class: 'font-mono text-xs block max-w-[280px] truncate',
        title: full,
      }, full)
    },
  },
  {
    id: 'status',
    header: () => h('div', { class: 'text-center' }, t('mcp.active')),
    cell: ({ row }) => h('div', { class: 'text-center' },
      h(Badge, { variant: row.original.is_active ? 'default' : 'secondary' },
        () => row.original.is_active ? 'ON' : 'OFF'),
    ),
  },
  {
    id: 'actions',
    header: () => h('div', { class: 'text-center' }, t('common.operation')),
    cell: ({ row }) => h('div', { class: 'flex gap-2 justify-center' }, [
      h(Button, {
        size: 'sm',
        variant: 'outline',
        onClick: () => openEditDialog(row.original),
      }, () => t('common.edit')),
      h(ConfirmPopover, {
        message: t('mcp.deleteConfirm'),
        onConfirm: () => handleDelete(row.original.id),
      }, {
        trigger: () => h(Button, {
          size: 'sm',
          variant: 'destructive',
        }, () => t('common.delete')),
      }),
    ]),
  },
])

async function loadList() {
  loading.value = true
  try {
    const { data } = await getBotsByBotIdMcp({
      path: { bot_id: props.botId },
      throwOnError: true,
    })
    items.value = data.items ?? []
  } catch (error) {
    toast.error(resolveErrorMessage(error, t('common.loadFailed')))
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingItem.value = null
  addDialogTab.value = 'single'
  importJson.value = ''
  connectionMode.value = 'stdio'
  formData.value = { name: '', command: '', url: '', cwd: '', transport: 'http', active: true }
  argsTags.value = []
  envTags.initFromObject(null)
  headerTags.initFromObject(null)
  formDialogOpen.value = true
}

function openEditDialog(item: McpItem) {
  editingItem.value = item
  const cfg = item.config ?? {}
  connectionMode.value = item.type === 'stdio' ? 'stdio' : 'remote'
  formData.value = {
    name: item.name,
    command: configValue(cfg, 'command'),
    url: configValue(cfg, 'url'),
    cwd: configValue(cfg, 'cwd'),
    transport: item.type === 'sse' ? 'sse' : 'http',
    active: !!item.is_active,
  }
  argsTags.value = configArray(cfg, 'args')
  envTags.initFromObject(configMap(cfg, 'env'))
  headerTags.initFromObject(configMap(cfg, 'headers'))
  editSyncFromJson = false
  syncFormToEditJson()
  formDialogOpen.value = true
}

function buildFormToEntry(): McpServerEntry | null {
  const d = formData.value
  const name = d.name.trim()
  if (!name) return null
  if (d.command.trim()) {
    const entry: McpServerEntry = {
      command: d.command.trim(),
      args: argsTags.value.length ? argsTags.value : undefined,
      cwd: d.cwd.trim() || undefined,
    }
    const env: Record<string, string> = {}
    envTags.tagList.value.forEach((tag) => {
      const [k, v] = tag.split(':')
      if (k && v) env[k] = v
    })
    if (Object.keys(env).length > 0) entry.env = env
    return entry
  }
  if (d.url.trim()) {
    const entry: McpServerEntry = {
      url: d.url.trim(),
      transport: d.transport === 'sse' ? 'sse' : undefined,
    }
    const headers: Record<string, string> = {}
    headerTags.tagList.value.forEach((tag) => {
      const [k, v] = tag.split(':')
      if (k && v) headers[k] = v
    })
    if (Object.keys(headers).length > 0) entry.headers = headers
    return entry
  }
  return null
}

function syncFormToEditJson() {
  if (editSyncFromJson) return
  const entry = buildFormToEntry()
  if (!entry) {
    editJson.value = ''
    return
  }
  const name = formData.value.name.trim()
  const mcpServers: Record<string, McpServerEntry> = { [name]: entry }
  editJson.value = JSON.stringify({ mcpServers }, null, 2)
}

function syncEditJsonToForm() {
  const raw = editJson.value.trim()
  if (!raw) return
  editSyncFromJson = true
  try {
    let parsed: { mcpServers?: Record<string, McpServerEntry> } = JSON.parse(raw)
    if (!parsed.mcpServers && typeof parsed === 'object' && !Array.isArray(parsed)) {
      parsed = { mcpServers: parsed as Record<string, McpServerEntry> }
    }
    const servers = parsed.mcpServers
    if (!servers || typeof servers !== 'object') {
      editSyncFromJson = false
      return
    }
    const entries = Object.entries(servers)
    const single = entries.length === 1 ? entries[0] : null
    if (!single) {
      editSyncFromJson = false
      return
    }
    const [name, e] = single
    if (e.command) {
      connectionMode.value = 'stdio'
      formData.value = {
        name,
        command: e.command ?? '',
        url: '',
        cwd: e.cwd ?? '',
        transport: 'http',
        active: formData.value.active,
      }
      argsTags.value = e.args ?? []
      envTags.initFromObject(e.env ?? null)
      headerTags.initFromObject(null)
    } else if (e.url) {
      connectionMode.value = 'remote'
      formData.value = {
        name,
        command: '',
        url: e.url ?? '',
        cwd: '',
        transport: e.transport === 'sse' ? 'sse' : 'http',
        active: formData.value.active,
      }
      argsTags.value = []
      envTags.initFromObject(null)
      headerTags.initFromObject(e.headers ?? null)
    }
  } catch {
    // ignore parse error
  }
  editSyncFromJson = false
}

function buildRequestBody() {
  const body: Record<string, unknown> = {
    name: formData.value.name.trim(),
    is_active: formData.value.active,
  }
  if (formData.value.command.trim()) {
    body.command = formData.value.command.trim()
    if (argsTags.value.length > 0) body.args = argsTags.value
    const env: Record<string, string> = {}
    envTags.tagList.value.forEach((tag) => {
      const [k, v] = tag.split(':')
      if (k && v) env[k] = v
    })
    if (Object.keys(env).length > 0) body.env = env
    if (formData.value.cwd.trim()) body.cwd = formData.value.cwd.trim()
  } else if (formData.value.url.trim()) {
    body.url = formData.value.url.trim()
    const headers: Record<string, string> = {}
    headerTags.tagList.value.forEach((tag) => {
      const [k, v] = tag.split(':')
      if (k && v) headers[k] = v
    })
    if (Object.keys(headers).length > 0) body.headers = headers
    if (formData.value.transport === 'sse') body.transport = 'sse'
  }
  return body
}

async function handleSubmit() {
  submitting.value = true
  try {
    const body = buildRequestBody()
    if (editingItem.value) {
      await putBotsByBotIdMcpById({
        path: { bot_id: props.botId, id: editingItem.value.id },
        body: body as any,
        throwOnError: true,
      })
    } else {
      await postBotsByBotIdMcp({
        path: { bot_id: props.botId },
        body: body as any,
        throwOnError: true,
      })
    }
    formDialogOpen.value = false
    await loadList()
    toast.success(editingItem.value ? t('mcp.updateSuccess') : t('mcp.createSuccess'))
  } catch (error) {
    toast.error(resolveErrorMessage(error, t('common.saveFailed')))
  } finally {
    submitting.value = false
  }
}

async function handleDelete(id: string) {
  try {
    await deleteBotsByBotIdMcpById({
      path: { bot_id: props.botId, id },
      throwOnError: true,
    })
    selectedIds.value = selectedIds.value.filter((x) => x !== id)
    await loadList()
    toast.success(t('mcp.deleteSuccess'))
  } catch (error) {
    toast.error(resolveErrorMessage(error, t('mcp.deleteFailed')))
  }
}

async function handleBatchDelete() {
  if (selectedIds.value.length === 0) return
  try {
    await postBotsByBotIdMcpOpsBatchDelete({
      path: { bot_id: props.botId },
      body: { ids: selectedIds.value },
      throwOnError: true,
    })
    selectedIds.value = []
    await loadList()
    toast.success(t('mcp.deleteSuccess'))
  } catch (error) {
    toast.error(resolveErrorMessage(error, t('mcp.deleteFailed')))
  }
}

function handleBatchExport() {
  const selected = items.value.filter((i) => selectedIds.value.includes(i.id))
  if (selected.length === 0) return
  const mcpServers: Record<string, McpServerEntry> = {}
  selected.forEach((item) => {
    mcpServers[item.name] = itemToExportEntry(item)
  })
  exportJson.value = JSON.stringify({ mcpServers }, null, 2)
  exportDialogOpen.value = true
}

async function handleImport() {
  importSubmitting.value = true
  try {
    let parsed = JSON.parse(importJson.value)
    if (!parsed.mcpServers && typeof parsed === 'object') {
      parsed = { mcpServers: parsed }
    }
    await client.put({
      url: `/bots/${props.botId}/mcp-ops/import`,
      body: parsed,
      throwOnError: true,
    })
    formDialogOpen.value = false
    importJson.value = ''
    await loadList()
    toast.success(t('mcp.importSuccess'))
  } catch (error) {
    toast.error(resolveErrorMessage(error, t('mcp.importFailed')))
  } finally {
    importSubmitting.value = false
  }
}

function handleCopyExport() {
  navigator.clipboard.writeText(exportJson.value)
  toast.success(t('common.copied'))
}

// --- Marketplace ---
const marketplaceExpanded = ref(false)
const marketplaceTab = ref<'smithery' | 'modelscope'>('smithery')
const mpQuery = ref('')
const mpSearching = ref(false)
const mpServers = ref<SmitheryServer[]>([])
const mpPage = ref(1)
const mpTotalPages = ref(0)
const mpTotalCount = ref(0)
const mpExpandedName = ref<string | null>(null)
const mpDetail = ref<SmitheryDetail | null>(null)
const mpDetailLoading = ref(false)
const mpInstallingName = ref<string | null>(null)
const mpInstalledNames = ref<Set<string>>(new Set())
const mpSearchError = ref(false)

// Search cache (30s TTL)
const mpCache = new Map<string, { data: any; ts: number }>()
const MP_CACHE_TTL = 30_000

function mpCacheKey(q: string, page: number) {
  return `${q}||${page}`
}

let mpDebounceTimer: ReturnType<typeof setTimeout> | null = null

function mpDebouncedSearch() {
  if (mpDebounceTimer) clearTimeout(mpDebounceTimer)
  mpDebounceTimer = setTimeout(() => {
    mpPage.value = 1
    mpSearch()
  }, 300)
}

async function mpSearch() {
  const cacheKey = mpCacheKey(mpQuery.value, mpPage.value)
  const cached = mpCache.get(cacheKey)
  if (cached && Date.now() - cached.ts < MP_CACHE_TTL) {
    const body = cached.data
    mpServers.value = body?.servers ?? []
    mpTotalPages.value = body?.pagination?.totalPages ?? 0
    mpTotalCount.value = body?.pagination?.totalCount ?? 0
    mpExpandedName.value = null
    mpDetail.value = null
    mpSearchError.value = false
    return
  }

  mpSearching.value = true
  mpExpandedName.value = null
  mpDetail.value = null
  try {
    const resp = await client.get({
      url: '/mcp-marketplace/search',
      query: { q: mpQuery.value || undefined, page: mpPage.value, pageSize: 12 },
    })
    const body = resp.data as { servers?: SmitheryServer[]; pagination?: { totalPages: number; totalCount: number } }
    mpServers.value = body?.servers ?? []
    mpTotalPages.value = body?.pagination?.totalPages ?? 0
    mpTotalCount.value = body?.pagination?.totalCount ?? 0
    mpSearchError.value = false
    mpCache.set(cacheKey, { data: body, ts: Date.now() })
  } catch {
    mpServers.value = []
    mpTotalPages.value = 0
    mpSearchError.value = true
  } finally {
    mpSearching.value = false
  }
}

async function mpToggleDetail(server: SmitheryServer) {
  if (mpExpandedName.value === server.qualifiedName) {
    mpExpandedName.value = null
    mpDetail.value = null
    return
  }
  mpExpandedName.value = server.qualifiedName
  mpDetail.value = null
  mpDetailLoading.value = true
  try {
    const resp = await client.get({
      url: '/mcp-marketplace/detail',
      query: { name: server.qualifiedName },
    })
    mpDetail.value = resp.data as SmitheryDetail
  } catch {
    toast.error(t('mcp.marketplace.detailFailed'))
    mpExpandedName.value = null
  } finally {
    mpDetailLoading.value = false
  }
}

function mpHasHttpConnection(detail: SmitheryDetail): boolean {
  return !!(detail.deploymentUrl || detail.connections?.some((c) => c.type === 'http'))
}

function mpPrefillStdioForm(detail: SmitheryDetail) {
  const stdioConn = detail.connections?.find((c) => c.type === 'stdio') as any
  editingItem.value = null
  addDialogTab.value = 'single'
  importJson.value = ''
  connectionMode.value = 'stdio'
  const slug = detail.qualifiedName.split('/').pop() || detail.displayName
  formData.value = {
    name: slug,
    command: stdioConn?.stdioFunction || stdioConn?.runtime || '',
    url: '',
    cwd: '',
    transport: 'http',
    active: true,
  }
  argsTags.value = []
  envTags.initFromObject(null)
  headerTags.initFromObject(null)
  formDialogOpen.value = true
}

async function mpInstall(detail: SmitheryDetail) {
  if (!mpHasHttpConnection(detail)) {
    mpPrefillStdioForm(detail)
    return
  }
  mpInstallingName.value = detail.qualifiedName
  try {
    const httpConn = detail.connections?.find((c) => c.type === 'http')
    const url = httpConn?.deploymentUrl || detail.deploymentUrl
    if (!url) {
      toast.error(t('mcp.marketplace.noConnectionInfo'))
      return
    }
    const slug = detail.qualifiedName.split('/').pop() || detail.displayName
    await postBotsByBotIdMcp({
      path: { bot_id: props.botId },
      body: { name: slug, url } as any,
      throwOnError: true,
    })
    mpInstalledNames.value.add(detail.qualifiedName)
    await loadList()
    toast.success(t('mcp.marketplace.installSuccess', { name: detail.displayName }))
  } catch (error) {
    toast.error(resolveErrorMessage(error, t('mcp.marketplace.installFailed')))
  } finally {
    mpInstallingName.value = null
  }
}

function mpNextPage() {
  if (mpPage.value < mpTotalPages.value) {
    mpPage.value++
    mpSearch()
  }
}

function mpPrevPage() {
  if (mpPage.value > 1) {
    mpPage.value--
    mpSearch()
  }
}

function mpIsInstalled(name: string): boolean {
  if (mpInstalledNames.value.has(name)) return true
  const slug = name.split('/').pop()
  return items.value.some((i) => {
    if (i.name === slug) return true
    const cfg = i.config ?? {}
    const itemUrl = configValue(cfg, 'url')
    if (itemUrl && itemUrl.includes(name.replace('/', '%2F'))) return true
    if (itemUrl && slug && itemUrl.toLowerCase().includes(slug.toLowerCase())) return true
    return false
  })
}

watch(() => props.botId, () => {
  if (props.botId) loadList()
}, { immediate: true })
</script>
