<script>
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import SettingsSidebar from "@/components/settings/SettingsSidebar.svelte";
    import { pageTitle } from "@/stores/app";
    import { addSuccessToast } from "@/stores/toasts";
    import { t } from "@/i18n";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";

    $pageTitle = $t("Embeddings");

    const EMBEDDING_APIS = ["openai-embeddings"];

    let original = {};
    let embedding = emptyConfig();
    let isLoading = false;
    let isSaving = false;

    $: hasChanges = JSON.stringify(original) !== JSON.stringify(embedding);
    $: allEmbeddingModels = embedding.providers
        .flatMap((p) => p.models.map((m) => m.providerModelId || m.name))
        .filter(Boolean);

    
    let isMetricsLoading = false;
    let metrics = null;

    async function loadMetrics() {
        isMetricsLoading = true;
        try {
            const result = await ApiClient.send("/api/vector/metrics", { method: "GET" });
            metrics = result?.metrics || null;
        } catch (err) {
            ApiClient.error(err);
        } finally {
            isMetricsLoading = false;
        }
    }

    loadMetrics();

    loadSettings();

    function emptyConfig() {
        return {
            enabled: false,
            defaultModel: "",
            providers: [],
        };
    }

    async function loadSettings() {
        isLoading = true;
        try {
            const result = (await ApiClient.settings.getAll()) || {};
            initSettings(result.agents?.embedding);
        } catch (err) {
            ApiClient.error(err);
        }
        isLoading = false;
    }

    function initSettings(data) {
        const cfg = Object.assign(emptyConfig(), data || {});
        cfg.providers = (cfg.providers || []).map((p) => ({
            id: p.id || "",
            vendor: p.vendor || "",
            api: p.api || "openai-embeddings",
            baseUrl: p.baseUrl || "",
            apiKey: p.apiKey || "",
            enabled: !!p.enabled,
            models: (p.models || []).map((m) => ({
                name: m.name || "",
                providerModelId: m.providerModelId || "",
                dimensions: m.dimensions || 1536,
                enabled: !!m.enabled,
            })),
        }));
        embedding = cfg;
        original = JSON.parse(JSON.stringify(cfg));
    }

    async function save() {
        if (isSaving || !hasChanges) return;
        isSaving = true;
        try {
            const current = (await ApiClient.settings.getAll()) || {};
            const agents = Object.assign({}, current.agents || {}, { embedding });
            const payload = CommonHelper.filterRedactedProps({ agents });
            const result = await ApiClient.settings.update(payload);
            initSettings(result.agents?.embedding);
            addSuccessToast($t("Successfully saved embedding settings."));
        } catch (err) {
            ApiClient.error(err);
        }
        isSaving = false;
    }

    function reset() {
        embedding = JSON.parse(JSON.stringify(original));
    }

    function addProvider() {
        embedding.providers = embedding.providers.concat({
            id: "",
            vendor: "openai",
            api: "openai-embeddings",
            baseUrl: "",
            apiKey: "",
            enabled: true,
            models: [],
        });
    }

    function removeProvider(idx) {
        embedding.providers = embedding.providers.filter((_, i) => i !== idx);
    }

    function addModel(pIdx) {
        embedding.providers[pIdx].models = embedding.providers[pIdx].models.concat({
            name: "",
            providerModelId: "",
            dimensions: 1536,
            enabled: true,
        });
        embedding = embedding;
    }

    function removeModel(pIdx, mIdx) {
        embedding.providers[pIdx].models = embedding.providers[pIdx].models.filter((_, i) => i !== mIdx);
        embedding = embedding;
    }
</script>

<SettingsSidebar />

<PageWrapper>
    <header class="page-header">
        <nav class="breadcrumbs">
            <div class="breadcrumb-item">{$t("Settings")}</div>
            <div class="breadcrumb-item">{$pageTitle}</div>
        </nav>
    </header>

    <div class="wrapper">
        <form class="panel" autocomplete="off" on:submit|preventDefault={save}>
            <div class="content m-b-sm txt-xl">
                <p>{$t("Configure embedding providers and the default vector model.")}</p>
            </div>

            {#if isLoading}
                <div class="loader" />
            {:else}
                <div class="em-row">
                    <label class="em-check">
                        <input type="checkbox" bind:checked={embedding.enabled} />
                        {$t("Enable embeddings")}
                    </label>
                </div>

                <div class="em-field">
                    <label>{$t("Default embedding model")}</label>
                    <select bind:value={embedding.defaultModel}>
                        <option value="">-</option>
                        {#each allEmbeddingModels as model}
                            <option value={model}>{model}</option>
                        {/each}
                    </select>
                </div>

                <hr />

                <div class="flex">
                    <h3 class="section-title">{$t("Embedding providers")}</h3>
                    <div class="flex-fill" />
                    <button type="button" class="btn btn-sm btn-transparent" on:click={addProvider}>
                        <i class="ri-add-line" /> <span class="txt">{$t("Add provider")}</span>
                    </button>
                </div>

                {#each embedding.providers as provider, pIdx (pIdx)}
                    <div class="em-provider">
                        <div class="em-provider-head">
                            <label class="em-check">
                                <input type="checkbox" bind:checked={provider.enabled} />
                                {$t("Enabled")}
                            </label>
                            <button type="button" class="btn btn-xs btn-transparent btn-hint" on:click={() => removeProvider(pIdx)}>
                                <i class="ri-delete-bin-line" />
                            </button>
                        </div>

                        <div class="em-row">
                            <div class="em-field">
                                <label>{$t("ID")}</label>
                                <input type="text" bind:value={provider.id} placeholder="openai-embedding" />
                            </div>
                            <div class="em-field">
                                <label>{$t("Vendor")}</label>
                                <input type="text" bind:value={provider.vendor} placeholder="openai" />
                            </div>
                        </div>

                        <div class="em-row">
                            <div class="em-field">
                                <label>{$t("API")}</label>
                                <select bind:value={provider.api}>
                                    {#each EMBEDDING_APIS as api}
                                        <option value={api}>{api}</option>
                                    {/each}
                                </select>
                            </div>
                            <div class="em-field" />
                        </div>

                        <div class="em-row">
                            <div class="em-field">
                                <label>{$t("Base URL")}</label>
                                <input type="text" bind:value={provider.baseUrl} placeholder="https://api.openai.com/v1" />
                            </div>
                            <div class="em-field">
                                <label>{$t("API key")}</label>
                                <input type="password" bind:value={provider.apiKey} placeholder="sk-... or env:OPENAI_API_KEY" />
                            </div>
                        </div>

                        <div class="flex m-t-sm">
                            <strong class="txt-sm">{$t("Models")}</strong>
                            <div class="flex-fill" />
                            <button type="button" class="btn btn-xs btn-transparent" on:click={() => addModel(pIdx)}>
                                <i class="ri-add-line" /> <span class="txt">{$t("Add model")}</span>
                            </button>
                        </div>

                        {#each provider.models as model, mIdx (mIdx)}
                            <div class="em-model">
                                <div class="em-row">
                                    <div class="em-field">
                                        <label>{$t("Name")}</label>
                                        <input type="text" bind:value={model.name} placeholder="text-embedding-3-small" />
                                    </div>
                                    <div class="em-field">
                                        <label>{$t("Provider model id")}</label>
                                        <input type="text" bind:value={model.providerModelId} placeholder="text-embedding-3-small" />
                                    </div>
                                    <div class="em-field">
                                        <label>{$t("Dimensions")}</label>
                                        <input type="number" bind:value={model.dimensions} placeholder="1536" />
                                    </div>
                                </div>
                                <div class="em-model-flags">
                                    <label><input type="checkbox" bind:checked={model.enabled} /> {$t("Enabled")}</label>
                                    <button type="button" class="btn btn-xs btn-transparent btn-hint" on:click={() => removeModel(pIdx, mIdx)}>
                                        <i class="ri-delete-bin-line" />
                                    </button>
                                </div>
                            </div>
                        {/each}
                    </div>
                {/each}

                {#if !embedding.providers.length}
                    <p class="txt-hint">{$t("No embedding providers configured yet.")}</p>
                {/if}

                <div class="flex m-t-base">
                    <div class="flex-fill" />
                    {#if hasChanges}
                        <button type="button" class="btn btn-transparent btn-hint" disabled={isSaving} on:click={reset}>
                            <span class="txt">{$t("Cancel")}</span>
                        </button>
                    {/if}
                    <button type="submit" class="btn btn-expanded" class:btn-loading={isSaving} disabled={!hasChanges || isSaving}>
                        <span class="txt">{$t("Save changes")}</span>
                    </button>
                </div>
            {/if}
        </form>

        <!-- Embedding Queue Status Panel -->
        <div class="panel m-t-base">
            <header class="section-header flex m-b-sm">
                <h3 class="section-title">{$t("Embedding Queue Status")}</h3>
                <div class="flex-fill" />
                <button type="button" class="btn btn-sm btn-transparent" on:click={loadMetrics} disabled={isMetricsLoading}>
                    <i class="ri-refresh-line" class:spin={isMetricsLoading} /> <span class="txt">{$t("Refresh")}</span>
                </button>
            </header>

            {#if isMetricsLoading && !metrics}
                <div class="loader" />
            {:else if metrics}
                <div class="metrics-grid m-b-base">
                    <div class="metric-card">
                        <div class="metric-label">{$t("Pending Tasks")}</div>
                        <div class="metric-value">{metrics.pendingEmbeddings ?? 0}</div>
                        <div class="metric-sub">{$t("Embeddings currently waiting in queue")}</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-label">{$t("Total Stored Vectors")}</div>
                        <div class="metric-value">{metrics.vectorEntries ?? 0}</div>
                        <div class="metric-sub">{$t("Vectors successfully generated")}</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-label">{$t("Active Model")}</div>
                        <div class="metric-value metric-value-sm">{metrics.embeddingModel || $t("Not Configured")}</div>
                        <div class="metric-sub">
                            {$t("Status")}: 
                            {#if metrics.embeddingReady}
                                <span class="label label-sm label-success">{$t("Ready")}</span>
                            {:else}
                                <span class="label label-sm label-warning">{$t("Not Ready")}</span>
                            {/if}
                        </div>
                    </div>
                </div>
            {:else}
                <p class="txt-hint">{$t("Failed to load queue metrics.")}</p>
            {/if}
        </div>
    </div>
</PageWrapper>

<style>
    .em-row {
        display: flex;
        gap: 12px;
        margin-bottom: 10px;
    }
    .em-field {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    .em-field label {
        font-size: 12px;
        opacity: 0.7;
    }
    .em-field input,
    .em-field select {
        padding: 6px 8px;
        border: 1px solid var(--baseAlt2Color, #e4e6eb);
        border-radius: 6px;
        font-size: 13px;
    }
    .em-check {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        font-size: 13px;
    }
    .em-provider {
        border: 1px solid var(--baseAlt2Color, #e4e6eb);
        border-radius: 8px;
        padding: 12px;
        margin-bottom: 12px;
        background: var(--baseAlt1Color, #f8f9fa);
    }
    .em-provider-head {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8px;
    }
    .em-model {
        border: 1px dashed var(--baseAlt2Color, #d0d3d9);
        border-radius: 6px;
        padding: 8px;
        margin-bottom: 8px;
        background: var(--baseColor, #fff);
    }
    .em-model-flags {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        align-items: center;
        font-size: 12px;
    }

    .metrics-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
        gap: 0.75rem;
        margin-top: 1rem;
    }
    .metric-card {
        background: var(--baseAlt1Color);
        border: 1px solid var(--baseAlt2Color);
        border-radius: 6px;
        padding: 1rem;
    }
    .metric-label {
        font-size: 0.8125rem;
        color: var(--txtHintColor);
        margin-bottom: 0.35rem;
    }
    .metric-value {
        font-size: 1.5rem;
        font-weight: 600;
        line-height: 1.2;
    }
    .metric-value-sm {
        font-size: 1rem;
        word-break: break-all;
    }
    .metric-sub {
        font-size: 0.75rem;
        color: var(--txtHintColor);
        margin-top: 0.5rem;
    }
</style>
