<script>
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import AgentChartPreview from "@/components/agents/AgentChartPreview.svelte";
    import { pageTitle } from "@/stores/app";
    import { addSuccessToast } from "@/stores/toasts";
    import { t } from "@/i18n";
    import ApiClient from "@/utils/ApiClient";

    $pageTitle = $t("AI Agents");

    // runtime + catalog
    let runtime = { enabled: false, providers: [], allowSchemaChange: false };
    let tools = [];

    // projects + sessions
    let projects = [];
    let selectedProject = "";
    let sessions = [];
    let activeSession = null;
    let messages = [];

    // composer
    let draft = "";
    let attachedImages = []; // [{mimeType, data, name}]
    let allowWrites = false;
    let isRunning = false;
    let lastTraces = [];
    let pendingApprovals = [];

    // per-session provider/model override (when creating)
    let newProvider = "";
    let newModel = "";

    // per-project agent config (§9.1)
    let projectConfig = null;

    init();

    async function init() {
        await Promise.all([loadRuntime(), loadTools(), loadProjects()]);
    }

    async function loadRuntime() {
        try {
            runtime = await ApiClient.send("/api/agents", { method: "GET" });
        } catch (err) {
            if (!err?.isAbort) console.warn(err);
        }
    }

    async function loadTools() {
        try {
            tools = (await ApiClient.send("/api/agents/tools", { method: "GET" })) || [];
        } catch (err) {
            if (!err?.isAbort) console.warn(err);
        }
    }

    async function loadProjects() {
        try {
            projects = await ApiClient.collection("project").getFullList(200, { sort: "-created" });
            if (!selectedProject && projects.length) {
                selectProject(projects[0].id);
            }
        } catch (err) {
            if (!err?.isAbort) console.warn(err);
        }
    }

    async function selectProject(id) {
        selectedProject = id;
        activeSession = null;
        messages = [];
        await Promise.all([loadSessions(), loadProjectConfig()]);
    }

    async function loadProjectConfig() {
        if (!selectedProject) {
            projectConfig = null;
            return;
        }
        try {
            projectConfig = await ApiClient.send(`/api/agents/projects/${selectedProject}/config`, { method: "GET" });
        } catch (err) {
            if (!err?.isAbort) console.warn(err);
        }
    }

    async function saveProjectConfig() {
        if (!selectedProject || !projectConfig) return;
        try {
            projectConfig = await ApiClient.send(`/api/agents/projects/${selectedProject}/config`, {
                method: "PUT",
                body: projectConfig,
            });
            addSuccessToast($t("Project agent config saved"));
        } catch (err) {
            ApiClient.error(err);
        }
    }

    async function loadSessions() {
        if (!selectedProject) {
            sessions = [];
            return;
        }
        try {
            sessions =
                (await ApiClient.send("/api/agents/sessions", {
                    method: "GET",
                    query: { project: selectedProject },
                })) || [];
        } catch (err) {
            if (!err?.isAbort) console.warn(err);
        }
    }

    async function createSession() {
        if (!selectedProject) return;
        try {
            const providerId = newProvider || runtime.defaultProvider || "";
            const provider = (runtime.providers || []).find((p) => p.id === providerId);
            const modelId =
                newModel ||
                provider?.defaultModel ||
                provider?.models?.find((m) => m.enabled && (m.providerModelId || m.name))?.providerModelId ||
                provider?.models?.find((m) => m.enabled && (m.providerModelId || m.name))?.name ||
                runtime.defaultModel ||
                "";
            const session = await ApiClient.send("/api/agents/sessions", {
                method: "POST",
                body: {
                    project: selectedProject,
                    provider: providerId,
                    model: modelId,
                },
            });
            await loadSessions();
            openSession(session);
        } catch (err) {
            ApiClient.error(err);
        }
    }

    async function openSession(session) {
        activeSession = session;
        lastTraces = [];
        pendingApprovals = [];
        try {
            const data = await ApiClient.send(`/api/agents/sessions/${session.id}`, { method: "GET" });
            activeSession = data.session || session;
            messages = data.messages || [];
        } catch (err) {
            if (!err?.isAbort) console.warn(err);
        }
    }

    function onImageSelected(e) {
        const files = Array.from(e.target.files || []);
        files.forEach((file) => {
            const reader = new FileReader();
            reader.onload = () => {
                const result = reader.result || "";
                const base64 = String(result).split(",").pop();
                attachedImages = attachedImages.concat({
                    mimeType: file.type || "image/png",
                    data: base64,
                    name: file.name,
                });
            };
            reader.readAsDataURL(file);
        });
        e.target.value = "";
    }

    function removeImage(idx) {
        attachedImages = attachedImages.filter((_, i) => i !== idx);
    }

    async function send(extraApprovedTools = []) {
        if (!activeSession || isRunning) return;
        if (!draft.trim() && !attachedImages.length && !extraApprovedTools.length) return;

        isRunning = true;
        const body = {
            content: draft,
            images: attachedImages.map((img) => ({ mimeType: img.mimeType, data: img.data })),
            allowWrites: allowWrites,
            approvedTools: extraApprovedTools,
        };

        try {
            const result = await ApiClient.send(`/api/agents/sessions/${activeSession.id}/run`, {
                method: "POST",
                body,
            });
            messages = result.messages || messages;
            lastTraces = result.traces || [];
            pendingApprovals = result.pendingApprovals || [];
            if (result.sessionName) {
                activeSession = { ...activeSession, name: result.sessionName };
                loadSessions();
            }
            draft = "";
            attachedImages = [];
        } catch (err) {
            ApiClient.error(err);
        } finally {
            isRunning = false;
        }
    }

    // Approve the pending write tools and resume the run (no new user message).
    async function approveAndResume() {
        const approvedTools = pendingApprovals.map((p) => p.tool);
        pendingApprovals = [];
        await send(approvedTools);
    }

    async function renameSession() {
        if (!activeSession) return;
        const name = prompt($t("New session name"), activeSession.name);
        if (!name) return;
        try {
            const updated = await ApiClient.send(`/api/agents/sessions/${activeSession.id}`, {
                method: "PATCH",
                body: { name },
            });
            activeSession = updated;
            loadSessions();
            addSuccessToast($t("Session renamed"));
        } catch (err) {
            ApiClient.error(err);
        }
    }

    function riskClass(risk) {
        if (risk === "high") return "label-danger";
        if (risk === "medium") return "label-warning";
        return "label-success";
    }

    // Parse the latest data.query trace into a chart/table preview (proposal §10.1).
    $: queryPreview = extractQueryPreview(lastTraces);
    function extractQueryPreview(traces) {
        for (let i = (traces || []).length - 1; i >= 0; i--) {
            const tr = traces[i];
            if (tr.tool !== "data.query" || !tr.result || tr.error) continue;
            try {
                const parsed = JSON.parse(tr.result);
                const data = parsed.data || {};
                const rows = data.items || data.Items || [];
                return { hint: parsed.chart || { type: "table" }, rows };
            } catch (e) {
                return null;
            }
        }
        return null;
    }

    $: activeProvider = newProvider || runtime.defaultProvider || "";
    $: providerModels = (runtime.providers || []).find((p) => p.id === activeProvider)?.models || [];
</script>

<PageWrapper class="full-page">
    <div class="agents-workspace">
        <!-- LEFT: projects + sessions -->
        <aside class="aw-left">
            <div class="aw-section">
                <div class="aw-section-title">{$t("Projects")}</div>
                <div class="aw-list">
                    {#each projects as project (project.id)}
                        <button
                            class="aw-list-item {selectedProject === project.id ? 'active' : ''}"
                            on:click={() => selectProject(project.id)}
                        >
                            <i class="ri-folder-2-line" />
                            <span>{project.name || project.id}</span>
                        </button>
                    {/each}
                    {#if !projects.length}
                        <div class="aw-empty">{$t("No projects")}</div>
                    {/if}
                </div>
            </div>

            <div class="aw-section">
                <div class="aw-section-title">
                    {$t("Sessions")}
                    <button class="btn btn-xs btn-transparent" on:click={createSession} disabled={!selectedProject}>
                        <i class="ri-add-line" />
                    </button>
                </div>
                <div class="aw-list">
                    {#each sessions as session (session.id)}
                        <button
                            class="aw-list-item {activeSession?.id === session.id ? 'active' : ''}"
                            on:click={() => openSession(session)}
                        >
                            <i class="ri-chat-3-line" />
                            <span>{session.name || session.id}</span>
                        </button>
                    {/each}
                    {#if selectedProject && !sessions.length}
                        <div class="aw-empty">{$t("No sessions yet")}</div>
                    {/if}
                </div>
            </div>
        </aside>

        <!-- CENTER: conversation -->
        <section class="aw-center">
            {#if !runtime.enabled}
                <div class="aw-banner">{$t("The agent runtime is disabled. Configure providers in settings.")}</div>
            {/if}

            {#if activeSession}
                <header class="aw-header">
                    <div class="aw-title">
                        <strong>{activeSession.name || activeSession.id}</strong>
                        <button class="btn btn-xs btn-transparent" on:click={renameSession}>
                            <i class="ri-pencil-line" />
                        </button>
                    </div>
                    <div class="aw-meta">
                        {activeSession.provider || runtime.defaultProvider} · {activeSession.model || runtime.defaultModel}
                    </div>
                </header>

                <div class="aw-conversation">
                    {#each messages as msg}
                        <div class="aw-msg aw-msg-{msg.role}">
                            <div class="aw-msg-role">{msg.role}</div>
                            <div class="aw-msg-content">
                                {msg.content}
                                {#if msg.images?.length}
                                    <div class="aw-msg-images">
                                        {#each msg.images as img}
                                            <img src={`data:${img.mimeType};base64,${img.data}`} alt="attachment" />
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                        </div>
                    {/each}

                    {#if isRunning}
                        <div class="aw-msg aw-msg-assistant"><div class="aw-msg-content">…</div></div>
                    {/if}
                </div>

                {#if queryPreview}
                    <div class="aw-query-preview">
                        <div class="aw-section-title">{$t("Query Result")}</div>
                        <AgentChartPreview hint={queryPreview.hint} rows={queryPreview.rows} />
                    </div>
                {/if}

                {#if pendingApprovals.length}
                        <div class="aw-approval">
                        <div class="aw-approval-title">
                            <i class="ri-shield-keyhole-line" />
                            {$t("Write approval required")}
                        </div>
                        <ul>
                            {#each pendingApprovals as p}
                                <li><span class="label {riskClass(p.risk)}">{p.risk}</span> {p.tool}</li>
                            {/each}
                        </ul>
                        <div class="aw-approval-actions">
                            <button class="btn btn-sm btn-success" on:click={approveAndResume} disabled={isRunning}>
                                {$t("Approve & continue")}
                            </button>
                            <button class="btn btn-sm btn-transparent" on:click={() => (pendingApprovals = [])}>
                                {$t("Deny")}
                            </button>
                        </div>
                    </div>
                {/if}

                <footer class="aw-composer">
                    {#if attachedImages.length}
                        <div class="aw-attachments">
                            {#each attachedImages as img, idx}
                                <div class="aw-attachment">
                                    <img src={`data:${img.mimeType};base64,${img.data}`} alt={img.name} />
                                    <button class="aw-attachment-remove" on:click={() => removeImage(idx)}>×</button>
                                </div>
                            {/each}
                        </div>
                    {/if}
                    <textarea
                        bind:value={draft}
                        placeholder={$t("Message the agent…")}
                        rows="2"
                        on:keydown={(e) => {
                            if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) send();
                        }}
                    />
                    <div class="aw-composer-actions">
                        <label class="btn btn-sm btn-transparent" title={$t("Attach image")}>
                            <i class="ri-image-add-line" />
                            <input type="file" accept="image/*" multiple hidden on:change={onImageSelected} />
                        </label>
                        <label class="aw-allow-writes">
                            <input type="checkbox" bind:checked={allowWrites} />
                            {$t("Allow writes")}
                        </label>
                        <button class="btn btn-sm btn-primary" on:click={() => send()} disabled={isRunning}>
                            {$t("Send")}
                        </button>
                    </div>
                </footer>
            {:else}
                <div class="aw-placeholder">{$t("Select or create a session to begin.")}</div>
            {/if}
        </section>

        <!-- RIGHT: inspector -->
        <aside class="aw-right">
            <div class="aw-section">
                <div class="aw-section-title">{$t("Provider")}</div>
                <select bind:value={newProvider} class="aw-select">
                    <option value="">{$t("Default")} ({runtime.defaultProvider || "-"})</option>
                    {#each runtime.providers || [] as p}
                        <option value={p.id}>{p.id} ({p.vendor})</option>
                    {/each}
                </select>
            </div>

            <div class="aw-section">
                <div class="aw-section-title">{$t("Model")}</div>
                <select bind:value={newModel} class="aw-select">
                    <option value="">{$t("Default")} ({runtime.defaultModel || "-"})</option>
                    {#each providerModels as m}
                        <option value={m.providerModelId || m.name}>
                            {m.name}{m.supportsVision ? " 👁" : ""}
                        </option>
                    {/each}
                </select>
            </div>

            <div class="aw-section">
                <div class="aw-section-title">{$t("Scope")}</div>
                <div class="aw-scope">project_id: <code>{selectedProject || "-"}</code></div>
                <div class="aw-scope">
                    {$t("Schema changes")}: {runtime.allowSchemaChange ? $t("allowed") : $t("locked")}
                </div>
            </div>

            {#if projectConfig}
                <div class="aw-section">
                    <div class="aw-section-title">{$t("Project agent config")}</div>
                    <div class="aw-pc-field">
                        <label>{$t("Default provider")}</label>
                        <select bind:value={projectConfig.defaultProvider} class="aw-select">
                            <option value="">{$t("Inherit")}</option>
                            {#each runtime.providers || [] as p}
                                <option value={p.id}>{p.id}</option>
                            {/each}
                        </select>
                    </div>
                    <div class="aw-pc-field">
                        <label>{$t("Default model")}</label>
                        <input class="aw-select" type="text" bind:value={projectConfig.defaultModel} placeholder={$t("Inherit")} />
                    </div>
                    <div class="aw-pc-field">
                        <label>{$t("Schema changes")}</label>
                        <select bind:value={projectConfig.allowSchemaChange} class="aw-select">
                            <option value="inherit">{$t("Inherit")}</option>
                            <option value="allow">{$t("Allow")}</option>
                            <option value="deny">{$t("Deny")}</option>
                        </select>
                    </div>
                    <div class="aw-pc-field">
                        <label>{$t("Approval policy")}</label>
                        <select bind:value={projectConfig.approvalPolicy} class="aw-select">
                            <option value="inherit">{$t("Inherit")}</option>
                            <option value="manual">{$t("Manual (require approval)")}</option>
                            <option value="auto">{$t("Auto-approve writes")}</option>
                        </select>
                    </div>
                    <button class="btn btn-sm btn-primary aw-pc-save" on:click={saveProjectConfig}>
                        {$t("Save project config")}
                    </button>
                </div>
            {/if}

            <div class="aw-section">
                <div class="aw-section-title">{$t("Tools")}</div>
                <div class="aw-tools">
                    {#each tools as tool}
                        <div class="aw-tool">
                            <span class="label {riskClass(tool.risk)}">{tool.category}</span>
                            <code>{tool.name}</code>
                            {#if tool.requiresApproval}<i class="ri-lock-line" title={$t("requires approval")} />{/if}
                        </div>
                    {/each}
                </div>
            </div>

            {#if lastTraces.length}
                <div class="aw-section">
                    <div class="aw-section-title">{$t("Last tool calls")}</div>
                    <div class="aw-traces">
                        {#each lastTraces as tr}
                            <div class="aw-trace">
                                <code>{tr.tool}</code>
                                {#if tr.error}<span class="label label-danger">error</span>{/if}
                            </div>
                        {/each}
                    </div>
                </div>
            {/if}
        </aside>
    </div>
</PageWrapper>

<style>
    .agents-workspace {
        display: grid;
        grid-template-columns: 220px 1fr 280px;
        gap: 12px;
        height: calc(100vh - 90px);
    }
    .aw-left,
    .aw-right {
        display: flex;
        flex-direction: column;
        gap: 16px;
        overflow-y: auto;
        padding: 8px;
        background: var(--baseAlt1Color, #f4f5f7);
        border-radius: 8px;
    }
    .aw-center {
        display: flex;
        flex-direction: column;
        background: var(--baseColor, #fff);
        border: 1px solid var(--baseAlt2Color, #e4e6eb);
        border-radius: 8px;
        overflow: hidden;
    }
    .aw-section-title {
        font-size: 12px;
        font-weight: 600;
        text-transform: uppercase;
        opacity: 0.7;
        margin-bottom: 6px;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    .aw-list {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }
    .aw-list-item {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 6px 8px;
        border: 0;
        background: transparent;
        border-radius: 6px;
        cursor: pointer;
        text-align: left;
        width: 100%;
        font-size: 13px;
    }
    .aw-list-item:hover {
        background: rgba(0, 0, 0, 0.05);
    }
    .aw-list-item.active {
        background: var(--primaryColor, #5b6ee1);
        color: #fff;
    }
    .aw-empty,
    .aw-placeholder {
        opacity: 0.5;
        font-size: 13px;
        padding: 12px;
    }
    .aw-placeholder {
        margin: auto;
    }
    .aw-banner {
        background: #fff3cd;
        color: #664d03;
        padding: 8px 12px;
        font-size: 13px;
    }
    .aw-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 10px 14px;
        border-bottom: 1px solid var(--baseAlt2Color, #e4e6eb);
    }
    .aw-meta {
        font-size: 12px;
        opacity: 0.6;
    }
    .aw-conversation {
        flex: 1;
        overflow-y: auto;
        padding: 14px;
        display: flex;
        flex-direction: column;
        gap: 10px;
    }
    .aw-msg {
        max-width: 80%;
        padding: 8px 12px;
        border-radius: 8px;
        font-size: 14px;
        white-space: pre-wrap;
        word-break: break-word;
    }
    .aw-msg-role {
        font-size: 10px;
        text-transform: uppercase;
        opacity: 0.5;
        margin-bottom: 2px;
    }
    .aw-msg-user {
        align-self: flex-end;
        background: var(--primaryColor, #5b6ee1);
        color: #fff;
    }
    .aw-msg-assistant {
        align-self: flex-start;
        background: var(--baseAlt1Color, #f0f1f4);
    }
    .aw-msg-tool {
        align-self: flex-start;
        background: #eef6ff;
        font-family: monospace;
        font-size: 12px;
    }
    .aw-msg-images img {
        max-width: 160px;
        border-radius: 6px;
        margin-top: 6px;
    }
    .aw-approval {
        margin: 0 14px 10px;
        padding: 10px 12px;
        border: 1px solid #ffc107;
        background: #fff8e1;
        border-radius: 8px;
    }
    .aw-approval-title {
        font-weight: 600;
        margin-bottom: 6px;
    }
    .aw-approval ul {
        margin: 0 0 8px;
        padding-left: 6px;
        list-style: none;
    }
    .aw-approval-actions {
        display: flex;
        gap: 8px;
    }
    .aw-composer {
        border-top: 1px solid var(--baseAlt2Color, #e4e6eb);
        padding: 10px 14px;
    }
    .aw-composer textarea {
        width: 100%;
        resize: vertical;
        border: 1px solid var(--baseAlt2Color, #e4e6eb);
        border-radius: 6px;
        padding: 8px;
        font-size: 14px;
    }
    .aw-composer-actions {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-top: 8px;
    }
    .aw-allow-writes {
        font-size: 13px;
        display: flex;
        align-items: center;
        gap: 4px;
        margin-left: auto;
    }
    .aw-attachments {
        display: flex;
        gap: 8px;
        margin-bottom: 8px;
        flex-wrap: wrap;
    }
    .aw-attachment {
        position: relative;
    }
    .aw-attachment img {
        height: 48px;
        border-radius: 6px;
    }
    .aw-attachment-remove {
        position: absolute;
        top: -6px;
        right: -6px;
        border: 0;
        border-radius: 50%;
        width: 18px;
        height: 18px;
        background: #000;
        color: #fff;
        cursor: pointer;
        line-height: 1;
    }
    .aw-select {
        width: 100%;
        padding: 6px;
        border-radius: 6px;
        border: 1px solid var(--baseAlt2Color, #e4e6eb);
        font-size: 13px;
    }
    .aw-scope {
        font-size: 12px;
        margin-bottom: 4px;
    }
    .aw-pc-field {
        display: flex;
        flex-direction: column;
        gap: 3px;
        margin-bottom: 8px;
    }
    .aw-pc-field label {
        font-size: 11px;
        opacity: 0.7;
    }
    .aw-pc-save {
        width: 100%;
        margin-top: 4px;
    }
    .aw-tools,
    .aw-traces {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    .aw-tool,
    .aw-trace {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;
    }
    .label {
        font-size: 10px;
        padding: 1px 6px;
        border-radius: 10px;
        text-transform: uppercase;
    }
    .label-danger {
        background: #f8d7da;
        color: #842029;
    }
    .label-warning {
        background: #fff3cd;
        color: #664d03;
    }
    .label-success {
        background: #d1e7dd;
        color: #0f5132;
    }
</style>
