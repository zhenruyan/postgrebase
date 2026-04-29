<script>
    import ProjectUpsertPanel from "@/components/projects/ProjectUpsertPanel.svelte";
    import CopyIcon from "@/components/base/CopyIcon.svelte";
    import FormattedDate from "@/components/base/FormattedDate.svelte";
    import HorizontalScroller from "@/components/base/HorizontalScroller.svelte";
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import RefreshButton from "@/components/base/RefreshButton.svelte";
    import Searchbar from "@/components/base/Searchbar.svelte";
    import SortHeader from "@/components/base/SortHeader.svelte";
    import { pageTitle } from "@/stores/app";
    import { t } from "@/i18n";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";
    import { querystring, replace } from "svelte-spa-router";

    $pageTitle = $t("Project Management");

    const queryParams = new URLSearchParams($querystring);

    let projectUpsertPanel;
    let projects = [];
    let isLoading = false;
    let filter = queryParams.get("filter") || "";
    let sort = queryParams.get("sort") || "-created";

    $: if (sort !== -1 && filter !== -1) {
        // keep listing params in sync
        const query = new URLSearchParams({ filter, sort }).toString();
        replace("/projects?" + query);

        loadProjects();
    }

    export function loadProjects() {
        isLoading = true;

        projects = []; // reset

        const normalizedFilter = CommonHelper.normalizeSearchFilter(filter, [
            "id",
            "name",
            "created",
            "updated",
        ]);

        return ApiClient.collection("project")
            .getFullList(200, {
                sort: sort || "-created",
                filter: normalizedFilter,
            })
            .then((result) => {
                projects = result;
                isLoading = false;
            })
            .catch((err) => {
                if (!err?.isAbort) {
                    isLoading = false;
                    console.warn(err);
                    clearList();
                    ApiClient.error(err, false);
                }
            });
    }

    function clearList() {
        projects = [];
    }
</script>

<PageWrapper>
    <header class="page-header">
        <nav class="breadcrumbs">
            <div class="breadcrumb-item">{$pageTitle}</div>
        </nav>

        <RefreshButton on:refresh={() => loadProjects()} />

        <div class="flex-fill" />

        <button type="button" class="btn btn-expanded" on:click={() => projectUpsertPanel?.show()}>
            <i class="ri-add-line" />
            <span class="txt">{$t("New Project")}</span>
        </button>
    </header>

    <Searchbar
        value={filter}
        placeholder={$t("Search projects...")}
        extraAutocompleteKeys={["name"]}
        on:submit={(e) => (filter = e.detail)}
    />
    <div class="clearfix m-b-base" />

    <HorizontalScroller class="table-wrapper">
        <table class="table" class:table-loading={isLoading}>
            <thead>
                <tr>
                    <SortHeader class="col-type-text" name="id" bind:sort>
                        <div class="col-header-content">
                            <i class={CommonHelper.getFieldTypeIcon("primary")} />
                            <span class="txt">{$t("ID")}</span>
                        </div>
                    </SortHeader>

                    <SortHeader class="col-type-text" name="name" bind:sort>
                        <div class="col-header-content">
                            <i class={CommonHelper.getFieldTypeIcon("text")} />
                            <span class="txt">{$t("Name")}</span>
                        </div>
                    </SortHeader>

                    <SortHeader class="col-type-date col-field-created" name="created" bind:sort>
                        <div class="col-header-content">
                            <i class={CommonHelper.getFieldTypeIcon("date")} />
                            <span class="txt">{$t("Created")}</span>
                        </div>
                    </SortHeader>

                    <SortHeader class="col-type-date col-field-updated" name="updated" bind:sort>
                        <div class="col-header-content">
                            <i class={CommonHelper.getFieldTypeIcon("date")} />
                            <span class="txt">{$t("Updated")}</span>
                        </div>
                    </SortHeader>

                    <th class="col-type-action min-width" />
                </tr>
            </thead>
            <tbody>
                {#each projects as project (project.id)}
                    <tr
                        tabindex="0"
                        class="row-handle"
                        on:click={() => projectUpsertPanel?.show(project)}
                        on:keydown={(e) => {
                            if (e.code === "Enter" || e.code === "Space") {
                                e.preventDefault();
                                projectUpsertPanel?.show(project);
                            }
                        }}
                    >
                        <td class="col-type-text col-field-id">
                            <div class="label">
                                <CopyIcon value={project.id} />
                                <span class="txt">{project.id}</span>
                            </div>
                        </td>

                        <td class="col-type-text">
                            <span class="txt txt-ellipsis" title={project.name}>
                                {project.name}
                            </span>
                        </td>

                        <td class="col-type-date col-field-created">
                            <FormattedDate date={project.created} />
                        </td>

                        <td class="col-type-date col-field-updated">
                            <FormattedDate date={project.updated} />
                        </td>

                        <td class="col-type-action min-width">
                            <i class="ri-arrow-right-line" />
                        </td>
                    </tr>
                {:else}
                    {#if isLoading}
                        <tr>
                            <td colspan="99" class="p-xs">
                                <span class="skeleton-loader m-0" />
                            </td>
                        </tr>
                    {:else}
                        <tr>
                            <td colspan="99" class="txt-center txt-hint p-xs">
                                <h6>No projects found.</h6>
                                {#if filter?.length}
                                    <button
                                        type="button"
                                        class="btn btn-hint btn-expanded m-t-sm"
                                        on:click={() => (filter = "")}
                                    >
                                        <span class="txt">Clear filters</span>
                                    </button>
                                {/if}
                            </td>
                        </tr>
                    {/if}
                {/each}
            </tbody>
        </table>
    </HorizontalScroller>

    {#if projects.length}
        <small class="block txt-hint txt-right m-t-sm">Showing {projects.length} of {projects.length}</small>
    {/if}
</PageWrapper>

<ProjectUpsertPanel bind:this={projectUpsertPanel} on:save={() => loadProjects()} on:delete={() => loadProjects()} />
