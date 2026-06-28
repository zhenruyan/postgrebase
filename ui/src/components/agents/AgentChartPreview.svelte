<script>
    import { onDestroy, afterUpdate } from "svelte";
    import Chart from "chart.js/auto";
    import { t } from "@/i18n";

    // hint: { type: "table"|"line"|"bar"|"pie"|"metric", xField, yFields: [] }
    export let hint = { type: "table" };
    export let rows = [];

    let mode = "table"; // table | chart
    let canvas;
    let chartInstance = null;

    const palette = ["#5b6ee1", "#22b8cf", "#f783ac", "#ffa94d", "#51cf66", "#845ef7"];

    $: columns = rows.length ? Object.keys(rows[0]) : [];
    $: chartType = hint?.type || "table";
    $: canChart = chartType !== "table" && rows.length > 0;
    // default to chart view when a chart is recommended
    $: if (canChart && mode === "table" && !userPickedMode) mode = "chart";

    let userPickedMode = false;
    function pick(m) {
        userPickedMode = true;
        mode = m;
    }

    function metricValue() {
        const f = (hint.yFields || [])[0];
        if (!f) return 0;
        let sum = 0;
        for (const r of rows) sum += Number(r[f]) || 0;
        return sum;
    }

    function buildConfig() {
        const x = hint.xField;
        const ys = hint.yFields || [];
        const labels = rows.map((r, i) => (x ? r[x] : i + 1));

        if (chartType === "pie") {
            const f = ys[0];
            return {
                type: "pie",
                data: {
                    labels,
                    datasets: [{ data: rows.map((r) => Number(r[f]) || 0), backgroundColor: palette }],
                },
                options: { responsive: true, maintainAspectRatio: false },
            };
        }

        return {
            type: chartType === "line" ? "line" : "bar",
            data: {
                labels,
                datasets: ys.map((f, idx) => ({
                    label: f,
                    data: rows.map((r) => Number(r[f]) || 0),
                    borderColor: palette[idx % palette.length],
                    backgroundColor: palette[idx % palette.length],
                    tension: 0.3,
                })),
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: { legend: { display: ys.length > 1 } },
            },
        };
    }

    function renderChart() {
        if (!canvas || mode !== "chart" || chartType === "metric" || !canChart) return;
        if (chartInstance) {
            chartInstance.destroy();
            chartInstance = null;
        }
        chartInstance = new Chart(canvas, buildConfig());
    }

    afterUpdate(renderChart);
    onDestroy(() => chartInstance && chartInstance.destroy());
</script>

<div class="qp">
    <div class="qp-tabs">
        <button class="qp-tab {mode === 'table' ? 'active' : ''}" on:click={() => pick("table")}>
            {$t("Table")}
        </button>
        {#if canChart}
            <button class="qp-tab {mode === 'chart' ? 'active' : ''}" on:click={() => pick("chart")}>
                {$t("Chart")} ({chartType})
            </button>
        {/if}
    </div>

    {#if mode === "table" || !canChart}
        <div class="qp-table-wrap">
            <table class="qp-table">
                <thead>
                    <tr>
                        {#each columns as col}<th>{col}</th>{/each}
                    </tr>
                </thead>
                <tbody>
                    {#each rows.slice(0, 50) as row}
                        <tr>
                            {#each columns as col}<td>{typeof row[col] === "object" ? JSON.stringify(row[col]) : row[col]}</td>{/each}
                        </tr>
                    {/each}
                </tbody>
            </table>
            {#if !rows.length}<div class="qp-empty">{$t("No rows")}</div>{/if}
        </div>
    {:else if chartType === "metric"}
        <div class="qp-metric">
            <div class="qp-metric-value">{metricValue()}</div>
            <div class="qp-metric-label">{(hint.yFields || [])[0]}</div>
        </div>
    {:else}
        <div class="qp-canvas-wrap"><canvas bind:this={canvas} /></div>
    {/if}
</div>

<style>
    .qp {
        border: 1px solid var(--baseAlt2Color, #e4e6eb);
        border-radius: 8px;
        overflow: hidden;
    }
    .qp-tabs {
        display: flex;
        gap: 4px;
        padding: 6px;
        background: var(--baseAlt1Color, #f4f5f7);
    }
    .qp-tab {
        border: 0;
        background: transparent;
        padding: 4px 10px;
        border-radius: 6px;
        cursor: pointer;
        font-size: 12px;
    }
    .qp-tab.active {
        background: var(--primaryColor, #5b6ee1);
        color: #fff;
    }
    .qp-table-wrap {
        max-height: 280px;
        overflow: auto;
    }
    .qp-table {
        width: 100%;
        border-collapse: collapse;
        font-size: 12px;
    }
    .qp-table th,
    .qp-table td {
        border-bottom: 1px solid var(--baseAlt2Color, #eee);
        padding: 4px 8px;
        text-align: left;
        white-space: nowrap;
    }
    .qp-canvas-wrap {
        height: 280px;
        padding: 10px;
    }
    .qp-metric {
        padding: 24px;
        text-align: center;
    }
    .qp-metric-value {
        font-size: 36px;
        font-weight: 700;
    }
    .qp-metric-label {
        opacity: 0.6;
        font-size: 13px;
    }
    .qp-empty {
        padding: 16px;
        opacity: 0.5;
        font-size: 13px;
    }
</style>
