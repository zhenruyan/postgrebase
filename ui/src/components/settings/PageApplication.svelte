<script>
    import tooltip from "@/actions/tooltip";
    import Field from "@/components/base/Field.svelte";
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import SettingsSidebar from "@/components/settings/SettingsSidebar.svelte";
    import { appName, hideControls, pageTitle } from "@/stores/app";
    import { addSuccessToast } from "@/stores/toasts";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";

    $pageTitle = "应用设置";

    let originalFormSettings = {};
    let formSettings = {};
    let isLoading = false;
    let isSaving = false;
    let initialHash = "";

    $: initialHash = JSON.stringify(originalFormSettings);

    $: hasChanges = initialHash != JSON.stringify(formSettings);

    loadSettings();

    async function loadSettings() {
        isLoading = true;

        try {
            const settings = (await ApiClient.settings.getAll()) || {};
            init(settings);
        } catch (err) {
            ApiClient.error(err);
        }

        isLoading = false;
    }

    async function save() {
        if (isSaving || !hasChanges) {
            return;
        }

        isSaving = true;

        try {
            const settings = await ApiClient.settings.update(CommonHelper.filterRedactedProps(formSettings));
            init(settings);
            addSuccessToast("Successfully saved application settings.");
        } catch (err) {
            ApiClient.error(err);
        }

        isSaving = false;
    }

    function init(settings = {}) {
        $appName = settings?.meta?.appName;
        $hideControls = !!settings?.meta?.hideControls;

        formSettings = {
            meta: settings?.meta || {},
            logs: settings?.logs || {},
        };

        originalFormSettings = JSON.parse(JSON.stringify(formSettings));
    }

    function reset() {
        formSettings = JSON.parse(JSON.stringify(originalFormSettings || {}));
    }
</script>

<SettingsSidebar />

<PageWrapper>
    <header class="page-header">
        <nav class="breadcrumbs">
            <div class="breadcrumb-item">系统管理</div>
            <div class="breadcrumb-item">应用设置</div>
        </nav>
    </header>

    <div class="wrapper">
        <form class="panel" autocomplete="off" on:submit|preventDefault={save}>
            {#if isLoading}
                <div class="loader" />
            {:else}
                <div class="grid">
                    <div class="col-lg-6">
                        <Field class="form-field required" name="meta.appName" let:uniqueId>
                            <label for={uniqueId}>应用名</label>
                            <input
                                type="text"
                                id={uniqueId}
                                required
                                bind:value={formSettings.meta.appName}
                            />
                        </Field>
                    </div>

                    <div class="col-lg-6">
                        <Field class="form-field required" name="meta.appUrl" let:uniqueId>
                            <label for={uniqueId}>应用地址 </label>
                            <input type="text" id={uniqueId} required bind:value={formSettings.meta.appUrl} />
                        </Field>
                    </div>

                    <Field class="form-field required" name="logs.maxDays" let:uniqueId>
                        <label for={uniqueId}>日志最大保留天数</label>
                        <input type="number" id={uniqueId} required bind:value={formSettings.logs.maxDays} />
                    </Field>

                    <Field class="form-field form-field-toggle" name="meta.hideControls" let:uniqueId>
                        <input type="checkbox" id={uniqueId} bind:checked={formSettings.meta.hideControls} />
                        <label for={uniqueId}>
                            <span class="txt">禁用修改表结构</span>
                            <i
                                class="ri-information-line link-hint"
                                use:tooltip={{
                                    text: `当你选择禁用后，表结构设置相关按钮会隐藏.`,
                                    position: "right",
                                }}
                            />
                        </label>
                    </Field>

                    <div class="col-lg-12 flex">
                        <div class="flex-fill" />
                        {#if hasChanges}
                            <button
                                type="button"
                                class="btn btn-transparent btn-hint"
                                disabled={isSaving}
                                on:click={() => reset()}
                            >
                                <span class="txt">取消</span>
                            </button>
                        {/if}
                        <button
                            type="submit"
                            class="btn btn-expanded"
                            class:btn-loading={isSaving}
                            disabled={!hasChanges || isSaving}
                            on:click={() => save()}
                        >
                            <span class="txt">确定</span>
                        </button>
                    </div>
                </div>
            {/if}
        </form>
    </div>
</PageWrapper>
