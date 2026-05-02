<script>
    import tooltip from "@/actions/tooltip";
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import S3Fields from "@/components/settings/S3Fields.svelte";
    import WebDAVFields from "@/components/settings/WebDAVFields.svelte";
    import SettingsSidebar from "@/components/settings/SettingsSidebar.svelte";
    import { t } from "@/i18n";
    import { pageTitle } from "@/stores/app";
    import { setErrors } from "@/stores/errors";
    import { addSuccessToast, addWarningToast, removeAllToasts } from "@/stores/toasts";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";
    import { slide } from "svelte/transition";

    $: $pageTitle = $t("File Storage");

    const testRequestKey = "s3_test_request";
    const webdavTestRequestKey = "webdav_test_request";

    let originalFormSettings = {};
    let formSettings = {};
    let isLoading = false;
    let isSaving = false;
    let isTesting = false;
    let testError = null;
    let isWebDAVTesting = false;
    let webdavTestError = null;

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
            ApiClient.cancelRequest(testRequestKey);
            ApiClient.cancelRequest(webdavTestRequestKey);
            const settings = await ApiClient.settings.update(CommonHelper.filterRedactedProps(formSettings));
            setErrors({});

            await init(settings);

            removeAllToasts();

            if (testError) {
                addWarningToast("Successfully saved but failed to establish S3 connection.");
            } else {
                addSuccessToast("Successfully saved files storage settings.");
            }
        } catch (err) {
            ApiClient.error(err);
        }

        isSaving = false;
    }

    async function init(settings = {}) {
        formSettings = {
            s3: settings?.s3 || {},
            webdav: settings?.webdav || {},
        };

        originalFormSettings = JSON.parse(JSON.stringify(formSettings));
    }

    async function reset() {
        formSettings = JSON.parse(JSON.stringify(originalFormSettings || {}));
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
        <form class="panel" autocomplete="off" on:submit|preventDefault={() => save()}>
            <div class="content txt-xl m-b-base">
                <p>{$t("By default files are stored on the local filesystem.")}</p>
                <p>{$t("Use MinIO or AWS S3 for more storage capabilities.")}</p>
            </div>

            {#if isLoading}
                <div class="loader" />
            {:else}
                <S3Fields
                    toggleLabel={$t("Use S3 storage")}
                    originalConfig={originalFormSettings.s3}
                    bind:config={formSettings.s3}
                    bind:isTesting
                    bind:testError
                >
                    {#if originalFormSettings.s3?.enabled != formSettings.s3.enabled}
                        <div transition:slide|local={{ duration: 150 }}>
                            <div class="alert alert-warning m-0">
                                <div class="icon">
                                    <i class="ri-error-warning-line" />
                                </div>
                                <div class="content">
                                    {$t("If files already exist, synchronize them from")}
                                    <strong>
                                        {originalFormSettings.s3?.enabled
                                            ? $t("S3 storage")
                                            : $t("local storage")}
                                    </strong>
                                    to the
                                    <strong
                                        >{formSettings.s3.enabled
                                            ? $t("S3 storage")
                                            : $t("local storage")}</strong
                                    >.
                                    <br />
                                    {$t("You can use tools like")}:
                                    <a
                                        href="https://github.com/rclone/rclone"
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        class="txt-bold"
                                    >
                                        rclone
                                    </a>,
                                    <a
                                        href="https://github.com/peak/s5cmd"
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        class="txt-bold"
                                    >
                                        s5cmd
                                    </a>, etc.
                                </div>
                            </div>
                            <div class="clearfix m-t-base" />
                        </div>
                    {/if}
                </S3Fields>

                <hr class="m-t-lg m-b-lg" />

                <WebDAVFields
                    toggleLabel={$t("Use WebDAV storage")}
                    configKey="webdav"
                    originalConfig={originalFormSettings.webdav}
                    bind:config={formSettings.webdav}
                    bind:isTesting={isWebDAVTesting}
                    bind:testError={webdavTestError}
                >
                    {#if originalFormSettings.webdav?.enabled != formSettings.webdav.enabled}
                        <div transition:slide|local={{ duration: 150 }}>
                            <div class="alert alert-warning m-0">
                                <div class="icon">
                                    <i class="ri-error-warning-line" />
                                </div>
                                <div class="content">
                                    {$t("If files already exist, synchronize them from")}
                                    <strong>
                                        {originalFormSettings.webdav?.enabled
                                            ? $t("WebDAV storage")
                                            : $t("other storage")}
                                    </strong>
                                    to the
                                    <strong
                                        >{formSettings.webdav.enabled
                                            ? $t("WebDAV storage")
                                            : $t("other storage")}</strong
                                    >.
                                    <br />
                                    {$t("You can use tools like")}:
                                    <a
                                        href="https://github.com/rclone/rclone"
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        class="txt-bold"
                                    >
                                        rclone
                                    </a>
                                </div>
                            </div>
                            <div class="clearfix m-t-base" />
                        </div>
                    {/if}
                </WebDAVFields>

                <div class="flex">
                    <div class="flex-fill" />

                    {#if formSettings.s3?.enabled && !hasChanges && !isSaving}
                        {#if isTesting}
                            <span class="loader loader-sm" />
                        {:else if testError}
                            <div
                                class="label label-sm label-warning entrance-right"
                                use:tooltip={testError.data?.message}
                            >
                                <i class="ri-error-warning-line txt-warning" />
                                <span class="txt">{$t("Failed to establish S3 connection")}</span>
                            </div>
                        {:else}
                            <div class="label label-sm label-success entrance-right">
                                <i class="ri-checkbox-circle-line txt-success" />
                                <span class="txt">{$t("S3 connected successfully")}</span>
                            </div>
                        {/if}
                    {/if}

                    {#if formSettings.webdav?.enabled && !hasChanges && !isSaving}
                        {#if isWebDAVTesting}
                            <span class="loader loader-sm" />
                        {:else if webdavTestError}
                            <div
                                class="label label-sm label-warning entrance-right"
                                use:tooltip={webdavTestError.data?.message}
                            >
                                <i class="ri-error-warning-line txt-warning" />
                                <span class="txt">{$t("Failed to establish WebDAV connection")}</span>
                            </div>
                        {:else}
                            <div class="label label-sm label-success entrance-right">
                                <i class="ri-checkbox-circle-line txt-success" />
                                <span class="txt">{$t("WebDAV connected successfully")}</span>
                            </div>
                        {/if}
                    {/if}

                    {#if hasChanges}
                        <button
                            type="button"
                            class="btn btn-transparent btn-hint"
                            disabled={isSaving}
                            on:click={() => reset()}
                        >
                            <span class="txt">{$t("Reset")}</span>
                        </button>
                    {/if}

                    <button
                        type="submit"
                        class="btn btn-expanded"
                        class:btn-loading={isSaving}
                        disabled={!hasChanges || isSaving}
                        on:click={() => save()}
                    >
                        <span class="txt">{$t("Save")}</span>
                    </button>
                </div>
            {/if}
        </form>
    </div>
</PageWrapper>
