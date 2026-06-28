<script>
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import SettingsSidebar from "@/components/settings/SettingsSidebar.svelte";
    import { pageTitle } from "@/stores/app";
    import { addSuccessToast } from "@/stores/toasts";
    import { t } from "@/i18n";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";

    $pageTitle = $t("AI Agents");

    const TOOL_NAMES = [
        "data.query",
        "data.get",
        "dataset.preview",
        "data.insert",
        "data.bulk_insert",
        "data.update",
        "data.delete",
        "schema.create_table",
        "schema.add_field",
        "schema.update_field",
        "schema.drop_field",
        "schema.create_index",
        "schema.set_relation",
    ];
    const PROVIDER_APIS = ["openai-chat", "openai-responses", "anthropic-messages", "google-gemini", "google-vertex"];

    let original = {};
    let agents = emptyConfig();
    let isLoading = false;
    let isSaving = false;

    $: hasChanges = JSON.stringify(original) !== JSON.stringify(agents);

    loadSettings();

    function emptyConfig() {
        return {
            enabled: false,
            defaultProvider: "",
            defaultModel: "",
            allowSchemaChange: false,
            allowedTools: [],
            providers: [],
        };
    }

    async function loadSettings() {
        isLoading = true;
        try {
            const result = (await ApiClient.settings.getAll()) || {};
            initSettings(result.agents);
        } catch (err) {
            ApiClient.error(err);
        }
        isLoading = false;
    }

    function initSettings(data) {
        const cfg = Object.assign(emptyConfig(), data || {});
        cfg.allowedTools = cfg.allowedTools || [];
        cfg.providers = (cfg.providers || []).map((p) => ({
            id: p.id || "",
            vendor: p.vendor || "",
            api: p.api || (p.vendor === "anthropic" ? "anthropic-messages" : "openai-chat"),
            baseUrl: p.baseUrl || "",
            apiKey: p.apiKey || "",
            enabled: !!p.enabled,
            defaultModel: p.defaultModel || "",
            models: (p.models || []).map((m) => ({
                name: m.name || "",
                providerModelId: m.providerModelId || "",
                enabled: !!m.enabled,
                supportsVision: !!m.supportsVision,
                supportsToolUse: !!m.supportsToolUse,
                supportsDocument: !!m.supportsDocument,
                embedding: !!m.embedding,
            })),
        }));
        agents = cfg;
        original = JSON.parse(JSON.stringify(cfg));
    }

    async function save() {
        if (isSaving || !hasChanges) return;
        isSaving = true;
        try {
            const payload = CommonHelper.filterRedactedProps({ agents });
            const result = await ApiClient.settings.update(payload);
            initSettings(result.agents);
            addSuccessToast($t("Successfully saved agent settings."));
        } catch (err) {
            ApiClient.error(err);
        }
        isSaving = false;
    }

    function reset() {
        agents = JSON.parse(JSON.stringify(original));
    }

    function addProvider() {
        agents.providers = agents.providers.concat({
            id: "",
            vendor: "openai",
            api: "openai-chat",
            baseUrl: "",
            apiKey: "",
            enabled: true,
            defaultModel: "",
            models: [],
        });
    }

    function removeProvider(idx) {
        agents.providers = agents.providers.filter((_, i) => i !== idx);
    }

    function addModel(pIdx) {
        agents.providers[pIdx].models = agents.providers[pIdx].models.concat({
            name: "",
            providerModelId: "",
            enabled: true,
            supportsVision: false,
            supportsToolUse: true,
            supportsDocument: false,
            embedding: false,
        });
        agents = agents;
    }

    function removeModel(pIdx, mIdx) {
        agents.providers[pIdx].models = agents.providers[pIdx].models.filter((_, i) => i !== mIdx);
        agents = agents;
    }

    function toggleTool(name) {
        if (agents.allowedTools.includes(name)) {
            agents.allowedTools = agents.allowedTools.filter((t) => t !== name);
        } else {
            agents.allowedTools = agents.allowedTools.concat(name);
        }
    }

    $: providerIds = agents.providers.map((p) => p.id).filter(Boolean);
    $: allModels = agents.providers.flatMap((p) => p.models.map((m) => m.providerModelId || m.name)).filter(Boolean);
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
                <p>{$t("Configure LLM providers, models and global agent policy.")}</p>
            </div>

            {#if isLoading}
                <div class="loader" />
            {:else}
                <!-- global policy -->
                <div class="ag-row">
                    <label class="ag-check">
                        <input type="checkbox" bind:checked={agents.enabled} />
                        {$t("Enable agent runtime")}
                    </label>
                    <label class="ag-check">
                        <input type="checkbox" bind:checked={agents.allowSchemaChange} />
                        {$t("Allow schema changes")}
                    </label>
                </div>

                <div class="ag-row">
                    <div class="ag-field">
                        <label>{$t("Default provider")}</label>
                        <select bind:value={agents.defaultProvider}>
                            <option value="">-</option>
                            {#each providerIds as id}
                                <option value={id}>{id}</option>
                            {/each}
                        </select>
                    </div>
                    <div class="ag-field">
                        <label>{$t("Default model")}</label>
                        <select bind:value={agents.defaultModel}>
                            <option value="">-</option>
                            {#each allModels as m}
                                <option value={m}>{m}</option>
                            {/each}
                        </select>
                    </div>
                </div>

                <hr />

                <!-- allowed tools -->
                <h3 class="section-title">{$t("Allowed tools")}</h3>
                <p class="txt-hint m-b-sm">{$t("Leave all unchecked to allow every tool.")}</p>
                <div class="ag-tools">
                    {#each TOOL_NAMES as name}
                        <label class="ag-tool-check">
                            <input
                                type="checkbox"
                                checked={agents.allowedTools.includes(name)}
                                on:change={() => toggleTool(name)}
                            />
                            <code>{name}</code>
                        </label>
                    {/each}
                </div>

                <hr />

                <!-- providers -->
                <div class="flex">
                    <h3 class="section-title">{$t("Providers")}</h3>
                    <div class="flex-fill" />
                    <button type="button" class="btn btn-sm btn-transparent" on:click={addProvider}>
                        <i class="ri-add-line" /> <span class="txt">{$t("Add provider")}</span>
                    </button>
                </div>

                {#each agents.providers as provider, pIdx (pIdx)}
                    <div class="ag-provider">
                        <div class="ag-provider-head">
                            <label class="ag-check">
                                <input type="checkbox" bind:checked={provider.enabled} />
                                {$t("Enabled")}
                            </label>
                            <button type="button" class="btn btn-xs btn-transparent btn-hint" on:click={() => removeProvider(pIdx)}>
                                <i class="ri-delete-bin-line" />
                            </button>
                        </div>
                        <div class="ag-row">
                            <div class="ag-field">
                                <label>{$t("ID")}</label>
                                <input type="text" bind:value={provider.id} placeholder="openai-main" />
                            </div>
                            <div class="ag-field">
                                <label>{$t("Vendor")}</label>
                                <input type="text" bind:value={provider.vendor} placeholder="openai" />
                            </div>
                        </div>
                        <div class="ag-row">
                            <div class="ag-field">
                                <label>{$t("API")}</label>
                                <select bind:value={provider.api}>
                                    {#each PROVIDER_APIS as api}
                                        <option value={api}>{api}</option>
                                    {/each}
                                </select>
                            </div>
                            <div class="ag-field" />
                        </div>
                        <div class="ag-row">
                            <div class="ag-field">
                                <label>{$t("Base URL")}</label>
                                <input type="text" bind:value={provider.baseUrl} placeholder="https://api.openai.com/v1" />
                            </div>
                            <div class="ag-field">
                                <label>{$t("API key")}</label>
                                <input type="password" bind:value={provider.apiKey} placeholder="sk-... or env:OPENAI_API_KEY" />
                            </div>
                        </div>
                        <div class="ag-field">
                            <label>{$t("Provider default model")}</label>
                            <input type="text" bind:value={provider.defaultModel} placeholder="gpt-4o" />
                        </div>

                        <div class="flex m-t-sm">
                            <strong class="txt-sm">{$t("Models")}</strong>
                            <div class="flex-fill" />
                            <button type="button" class="btn btn-xs btn-transparent" on:click={() => addModel(pIdx)}>
                                <i class="ri-add-line" /> <span class="txt">{$t("Add model")}</span>
                            </button>
                        </div>

                        {#each provider.models as model, mIdx (mIdx)}
                            <div class="ag-model">
                                <div class="ag-row">
                                    <div class="ag-field">
                                        <label>{$t("Name")}</label>
                                        <input type="text" bind:value={model.name} placeholder="gpt-4o" />
                                    </div>
                                    <div class="ag-field">
                                        <label>{$t("Provider model id")}</label>
                                        <input type="text" bind:value={model.providerModelId} placeholder="gpt-4o" />
                                    </div>
                                </div>
                                <div class="ag-model-flags">
                                    <label><input type="checkbox" bind:checked={model.enabled} /> {$t("Enabled")}</label>
                                    <label><input type="checkbox" bind:checked={model.supportsVision} /> {$t("Vision")}</label>
                                    <label><input type="checkbox" bind:checked={model.supportsToolUse} /> {$t("Tools")}</label>
                                    <label><input type="checkbox" bind:checked={model.embedding} /> {$t("Embedding")}</label>
                                    <button type="button" class="btn btn-xs btn-transparent btn-hint" on:click={() => removeModel(pIdx, mIdx)}>
                                        <i class="ri-delete-bin-line" />
                                    </button>
                                </div>
                            </div>
                        {/each}
                    </div>
                {/each}

                {#if !agents.providers.length}
                    <p class="txt-hint">{$t("No providers configured yet.")}</p>
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
    </div>
</PageWrapper>

<style>
    .ag-row {
        display: flex;
        gap: 12px;
        margin-bottom: 10px;
    }
    .ag-field {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    .ag-field label {
        font-size: 12px;
        opacity: 0.7;
    }
    .ag-field input,
    .ag-field select {
        padding: 6px 8px;
        border: 1px solid var(--baseAlt2Color, #e4e6eb);
        border-radius: 6px;
        font-size: 13px;
    }
    .ag-check,
    .ag-tool-check {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        font-size: 13px;
    }
    .ag-tools {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
        gap: 6px;
        margin-bottom: 8px;
    }
    .ag-provider {
        border: 1px solid var(--baseAlt2Color, #e4e6eb);
        border-radius: 8px;
        padding: 12px;
        margin-bottom: 12px;
        background: var(--baseAlt1Color, #f8f9fa);
    }
    .ag-provider-head {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8px;
    }
    .ag-model {
        border: 1px dashed var(--baseAlt2Color, #d0d3d9);
        border-radius: 6px;
        padding: 8px;
        margin-bottom: 8px;
        background: var(--baseColor, #fff);
    }
    .ag-model-flags {
        display: flex;
        flex-wrap: wrap;
        gap: 12px;
        align-items: center;
        font-size: 12px;
    }
</style>
