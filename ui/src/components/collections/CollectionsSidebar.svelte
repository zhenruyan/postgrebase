<script>
    import CollectionUpsertPanel from "@/components/collections/CollectionUpsertPanel.svelte";
    import { hideControls } from "@/stores/app";
    import { activeCollection, collections, isCollectionsLoading } from "@/stores/collections";
    import { admin } from "@/stores/admin";
    import { t } from "@/i18n";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";
    import { link } from "svelte-spa-router";

    let collectionPanel;
    let searchTerm = "";
    let projects = [];
    let selectedProjectId = ""; // empty for all

    async function loadProjects() {
        try {
            projects = await ApiClient.collection("project").getFullList(200, {
                sort: "name",
                $autoCancel: false, // 禁用自动取消，防止响应式触发导致的并发请求失败
            });
        } catch (err) {
            if (!err?.isAbort) {
                console.warn("Failed to load projects:", err);
            }
        }
    }

    $: if ($admin?.id || $collections) {
        loadProjects();
    }

    $: if ($collections) {
        scrollIntoView();
    }

    $: normalizedSearch = searchTerm.replace(/\s+/g, "").toLowerCase();

    $: hasSearch = searchTerm !== "";

    $: filtered = $collections.filter((collection) => {
        const name = collection.name.replace(/\s+/g, "").toLowerCase();
        const displayName = (collection.displayName || "").replace(/\s+/g, "").toLowerCase();
        const matchesSearch =
            collection.id == searchTerm ||
            name.includes(normalizedSearch) ||
            displayName.includes(normalizedSearch);

        if (!matchesSearch) {
            return false;
        }

        if (selectedProjectId === "unassigned") {
            return !collection.project || !projects.find((p) => p.id === collection.project);
        }

        if (selectedProjectId) {
            return collection.project === selectedProjectId;
        }

        return true;
    });

    function selectCollection(collection) {
        $activeCollection = collection;
    }

    function scrollIntoView() {
        setTimeout(() => {
            const activeItem = document.querySelector(".collection-sidebar .sidebar-list-item.active");
            if (activeItem) {
                activeItem?.scrollIntoView({ block: "nearest" });
            }
        }, 0);
    }
</script>

<aside class="page-sidebar collection-sidebar">
    <header class="sidebar-header">
        <div class="form-field search" class:active={hasSearch}>
            <div class="form-field-addon">
                <button
                    type="button"
                    class="btn btn-xs btn-transparent btn-circle btn-clear"
                    class:hidden={!hasSearch}
                    on:click={() => (searchTerm = "")}
                >
                    <i class="ri-close-line" />
                </button>
            </div>
            <input type="text" placeholder={$t("Search all tables...")} bind:value={searchTerm} />
        </div>

        <div class="project-filter-container m-t-10">
            <div class="form-field select-field m-0">
                <i class="ri-folder-line field-icon" />
                <select bind:value={selectedProjectId} class="project-select">
                    <option value="">{$t("All Projects")}</option>
                    {#each projects as project (project.id)}
                        <option value={project.id}>{project.name}</option>
                    {/each}
                    <option value="unassigned">{$t("Unassigned Projects")}</option>
                </select>
                <i class="ri-arrow-down-s-line arrow-icon" />
            </div>
        </div>
    </header>

    <hr class="m-t-5 m-b-xs" />

    <div
        class="sidebar-content"
        class:fade={$isCollectionsLoading}
        class:sidebar-content-compact={filtered.length > 20}
    >
        {#each projects as project}
            {#if filtered.filter(c => c.project === project.id).length > 0 || (!hasSearch && projects.length > 0)}
                {@const projectCollections = filtered.filter(c => c.project === project.id)}
                <div class="project-group m-b-sm">
                    <div class="project-title txt-hint txt-sm p-l-sm p-r-sm m-b-xs flex">
                        <i class="ri-folder-line m-r-5" />
                        <span class="txt-ellipsis">{project.name}</span>
                    </div>
                    {#each projectCollections as collection (collection.id)}
                        <a
                            href="/collections?collectionId={collection.id}"
                            class="sidebar-list-item"
                            title={collection.displayName || collection.name}
                            class:active={$activeCollection?.id === collection.id}
                            use:link
                        >
                            <i class={CommonHelper.getCollectionTypeIcon(collection.type)} />
                            <span class="txt">{collection.displayName || collection.name}</span>
                        </a>
                    {/each}
                </div>
            {/if}
        {/each}

        {#if filtered.filter(c => !c.project || !projects.find(p => p.id === c.project)).length > 0}
            {@const unassignedCollections = filtered.filter(c => !c.project || !projects.find(p => p.id === c.project))}
            {#if projects.length > 0}
                <div class="project-title txt-hint txt-sm p-l-sm p-r-sm m-b-xs m-t-sm flex">
                    <i class="ri-folder-unknow-line m-r-5" />
                    <span class="txt-ellipsis">{$t("Unassigned Projects")}</span>
                </div>
            {/if}
            {#each unassignedCollections as collection (collection.id)}
                <a
                    href="/collections?collectionId={collection.id}"
                    class="sidebar-list-item"
                    title={collection.displayName || collection.name}
                    class:active={$activeCollection?.id === collection.id}
                    use:link
                >
                    <i class={CommonHelper.getCollectionTypeIcon(collection.type)} />
                    <span class="txt">{collection.displayName || collection.name}</span>
                </a>
            {/each}
        {/if}

        {#if filtered.length === 0 && normalizedSearch.length}
            <p class="txt-hint m-t-10 m-b-10 txt-center">未搜索到表</p>
        {/if}
    </div>

    {#if !$hideControls}
        <footer class="sidebar-footer">
            <button type="button" class="btn btn-block btn-outline" on:click={() => collectionPanel?.show()}>
                <i class="ri-add-line" />
                <span class="txt">{$t("New Table")}</span>
            </button>
        </footer>
    {/if}
</aside>

<CollectionUpsertPanel
    bind:this={collectionPanel}
    on:save={(e) => {
        if (e.detail?.isNew && e.detail.collection) {
            selectCollection(e.detail.collection);
        }
    }}
/>

<style>
    .project-title {
        font-weight: bold;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        opacity: 0.7;
        align-items: center;
    }
    .project-filter-container {
        padding: 0 5px;
    }
    .select-field {
        position: relative;
        display: flex;
        align-items: center;
    }
    .field-icon {
        position: absolute;
        left: 10px;
        z-index: 1;
        pointer-events: none;
        opacity: 0.5;
    }
    .arrow-icon {
        position: absolute;
        right: 10px;
        pointer-events: none;
        opacity: 0.5;
    }
    .project-select {
        padding-left: 32px !important;
        appearance: none;
        cursor: pointer;
        background: var(--baseAltColor);
        border: 1px solid var(--baseColor);
        border-radius: var(--borderRadius);
        transition: border-color 0.2s;
    }
    .project-select:hover {
        border-color: var(--primaryColor);
    }
</style>
