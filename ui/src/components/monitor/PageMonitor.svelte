<script>
    import { onMount, onDestroy } from "svelte";
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import RefreshButton from "@/components/base/RefreshButton.svelte";
    import { pageTitle } from "@/stores/app";
    import { t } from "@/i18n";
    import ApiClient from "@/utils/ApiClient";

    $pageTitle = $t("Monitor");

    let metrics = null;
    let cluster = null;
    let isLoading = false;
    let autoRefresh = true;
    let timer = null;

    $: isCluster = cluster?.view?.mode === "cluster";
    $: members = cluster?.view?.members || [];
    $: cacheBackendLabel = metrics?.cacheBackend === "redis" || (!metrics?.cacheBackend && metrics?.redisEnabled)
        ? "Redis"
        : $t("Memory Cache");

    async function load(showSpinner = true) {
        if (showSpinner) {
            isLoading = true;
        }
        try {
            const [m, c] = await Promise.all([
                ApiClient.send("/api/vector/metrics", { method: "GET" }),
                ApiClient.send("/api/vector/cluster", { method: "GET" }),
            ]);
            metrics = m?.metrics || null;
            cluster = c || null;
        } catch (err) {
            if (!err?.isAbort) {
                console.warn(err);
                ApiClient.error(err, false);
            }
        } finally {
            isLoading = false;
        }
    }

    function formatBytes(bytes) {
        if (!bytes && bytes !== 0) return "-";
        const units = ["B", "KB", "MB", "GB", "TB"];
        let value = bytes;
        let i = 0;
        while (value >= 1024 && i < units.length - 1) {
            value /= 1024;
            i++;
        }
        return value.toFixed(value >= 100 || i === 0 ? 0 : 1) + " " + units[i];
    }

    function formatUptime(seconds) {
        if (!seconds && seconds !== 0) return "-";
        const d = Math.floor(seconds / 86400);
        const h = Math.floor((seconds % 86400) / 3600);
        const m = Math.floor((seconds % 3600) / 60);
        const s = Math.floor(seconds % 60);
        if (d > 0) return `${d}d ${h}h ${m}m`;
        if (h > 0) return `${h}h ${m}m ${s}s`;
        if (m > 0) return `${m}m ${s}s`;
        return `${s}s`;
    }

    function toggleAutoRefresh() {
        autoRefresh = !autoRefresh;
        setupTimer();
    }

    function setupTimer() {
        if (timer) {
            clearInterval(timer);
            timer = null;
        }
        if (autoRefresh) {
            timer = setInterval(() => load(false), 5000);
        }
    }

    onMount(() => {
        load();
        setupTimer();
    });

    onDestroy(() => {
        if (timer) {
            clearInterval(timer);
        }
    });
</script>

<PageWrapper>
    <header class="page-header">
        <nav class="breadcrumbs">
            <div class="breadcrumb-item">{$t("Monitor")}</div>
        </nav>

        <RefreshButton on:refresh={() => load()} />

        <div class="flex-fill" />

        <span class="label {isCluster ? 'label-warning' : 'label-success'}">
            {isCluster ? $t("Cluster Mode") : $t("Standalone Mode")}
        </span>

        <button type="button" class="btn btn-sm btn-hint btn-border m-l-sm" on:click={toggleAutoRefresh}>
            <i class={autoRefresh ? "ri-pause-line" : "ri-play-line"} />
            <span class="txt">{autoRefresh ? $t("Pause Auto Refresh") : $t("Resume Auto Refresh")}</span>
        </button>
    </header>

    {#if metrics}
        <!-- single-node base metrics -->
        <div class="metrics-grid">
            <div class="metric-card">
                <div class="metric-label">{$t("Service Status")}</div>
                <div class="metric-value">
                    {#if metrics.online}
                        <span class="dot dot-online" /> {$t("Online")}
                    {:else}
                        <span class="dot dot-offline" /> {$t("Offline")}
                    {/if}
                </div>
                <div class="metric-sub">{$t("Uptime")} {formatUptime(metrics.uptimeSeconds)}</div>
            </div>

            <div class="metric-card">
                <div class="metric-label">{$t("Memory Usage")}</div>
                <div class="metric-value">{formatBytes(metrics.memAllocBytes)}</div>
                <div class="metric-sub">{$t("System Reserved")} {formatBytes(metrics.memSysBytes)}</div>
            </div>

            <div class="metric-card">
                <div class="metric-label">Goroutines / CPU</div>
                <div class="metric-value">{metrics.goroutines}</div>
                <div class="metric-sub">{metrics.numCpu} {$t("cores")} · GC {metrics.gcCount}</div>
            </div>

            <div class="metric-card">
                <div class="metric-label">{$t("Vector Index Count")}</div>
                <div class="metric-value">{metrics.vectorEntries}</div>
                <div class="metric-sub">{$t("Backend")} {metrics.backend}</div>
            </div>

            <div class="metric-card">
                <div class="metric-label">{$t("Embedding Queue")}</div>
                <div class="metric-value">{metrics.pendingEmbeddings}</div>
                <div class="metric-sub">
                    {$t("Model")} {metrics.embeddingModel || $t("Not Configured")}
                    {#if metrics.embeddingReady}
                        <span class="label label-sm label-success">{$t("Ready")}</span>
                    {:else}
                        <span class="label label-sm label-warning">{$t("Not Ready")}</span>
                    {/if}
                </div>
            </div>

            <div class="metric-card">
                <div class="metric-label">{$t("Cache Items")}</div>
                <div class="metric-value">{metrics.cacheItems}</div>
                <div class="metric-sub">
                    <span class="label label-sm {cacheBackendLabel === 'Redis' ? 'label-warning' : 'label-success'}">
                        {cacheBackendLabel}
                    </span>
                </div>
            </div>

            <div class="metric-card">
                <div class="metric-label">{$t("Primary DB Driver")}</div>
                <div class="metric-value">{metrics.dataDriver}</div>
                <div class="metric-sub">{$t("Node")} {metrics.nodeId}</div>
            </div>
        </div>
    {:else if isLoading}
        <div class="block txt-center p-xl txt-hint">
            <span class="loader" /> {$t("Loading")}...
        </div>
    {:else}
        <div class="block txt-center p-xl txt-hint">
            <i class="ri-error-warning-line" style="font-size: 2rem; opacity: 0.3;" />
            <h6 class="m-t-sm">{$t("Vector runtime is not enabled")}</h6>
        </div>
    {/if}

    <!-- cluster view (only when in cluster mode) -->
    {#if isCluster}
        <div class="section-title m-t-lg">
            <i class="ri-server-line" />
            <span>{$t("Cluster Status")}</span>
        </div>

        <div class="metrics-grid">
            <div class="metric-card">
                <div class="metric-label">{$t("Leader Node")}</div>
                <div class="metric-value metric-value-sm">{cluster.view.leaderId || $t("Electing")}</div>
                <div class="metric-sub">{cluster.view.isLeader ? $t("This node is Leader") : $t("This node is Follower")}</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">{$t("Raft Term")}</div>
                <div class="metric-value">{cluster.view.term}</div>
                <div class="metric-sub">{$t("Node")} {cluster.view.nodeId}</div>
            </div>
            <div class="metric-card">
                <div class="metric-label">{$t("Total Nodes")}</div>
                <div class="metric-value">{members.length}</div>
                <div class="metric-sub">
                    {members.filter((m) => m.alive).length} {$t("online")} ·
                    {members.filter((m) => !m.alive).length} {$t("offline")}
                </div>
            </div>
        </div>

        <div class="section-subtitle m-t-base">{$t("Node List")}</div>
        <table class="table">
            <thead>
                <tr>
                    <th>{$t("Address")}</th>
                    <th>{$t("Node ID")}</th>
                    <th>{$t("Role")}</th>
                    <th>{$t("Health")}</th>
                    <th>Term</th>
                    <th>{$t("Last Heartbeat")}</th>
                </tr>
            </thead>
            <tbody>
                {#each members as member (member.address)}
                    <tr class:row-self={member.isSelf}>
                        <td>
                            <strong>{member.address}</strong>
                            {#if member.isSelf}
                                <span class="label label-sm">{$t("This Node")}</span>
                            {/if}
                        </td>
                        <td><code>{member.nodeId || "-"}</code></td>
                        <td>
                            {#if member.isLeader}
                                <span class="label label-warning">Leader</span>
                            {:else}
                                <span class="label">Follower</span>
                            {/if}
                        </td>
                        <td>
                            {#if member.alive}
                                <span class="dot dot-online" /> {$t("Online")}
                            {:else}
                                <span class="dot dot-offline" /> {$t("Offline")}
                            {/if}
                        </td>
                        <td>{member.term}</td>
                        <td>
                            {#if member.lastSeen}
                                {new Date(member.lastSeen).toLocaleString()}
                            {:else}
                                -
                            {/if}
                        </td>
                    </tr>
                {/each}
            </tbody>
        </table>
    {/if}
</PageWrapper>

<style>
    .metrics-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
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
        margin-top: 0.35rem;
    }

    .dot {
        display: inline-block;
        width: 8px;
        height: 8px;
        border-radius: 50%;
        margin-right: 4px;
    }
    .dot-online {
        background: rgb(34, 197, 94);
    }
    .dot-offline {
        background: rgb(239, 68, 68);
    }

    .section-title {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        font-size: 1.1rem;
        font-weight: 600;
    }

    .section-subtitle {
        font-weight: 600;
        margin-bottom: 0.5rem;
    }

    .label-sm {
        font-size: 0.7rem;
        padding: 0.1rem 0.35rem;
        margin-left: 0.35rem;
    }

    .row-self {
        background: var(--baseAlt1Color);
    }
</style>
