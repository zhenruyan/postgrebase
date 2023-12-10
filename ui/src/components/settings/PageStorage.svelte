<script>
    import tooltip from "@/actions/tooltip";
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import S3Fields from "@/components/settings/S3Fields.svelte";
    import SettingsSidebar from "@/components/settings/SettingsSidebar.svelte";
    import { pageTitle } from "@/stores/app";
    import { setErrors } from "@/stores/errors";
    import { addSuccessToast, addWarningToast, removeAllToasts } from "@/stores/toasts";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";
    import { slide } from "svelte/transition";

    $pageTitle = "文件存储";

    const testRequestKey = "s3_test_request";

    let originalFormSettings = {};
    let formSettings = {};
    let isLoading = false;
    let isSaving = false;
    let isTesting = false;
    let testError = null;

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
            <div class="breadcrumb-item">设置</div>
            <div class="breadcrumb-item">{$pageTitle}</div>
        </nav>
    </header>

    <div class="wrapper">
        <form class="panel" autocomplete="off" on:submit|preventDefault={() => save()}>
            <div class="content txt-xl m-b-base">
                <p>默认情况下会直接存储到本地目录</p>
                <p>
                  如果想要更多功能可以使用minio或者aws s3 
                </p>
            </div>

            {#if isLoading}
                <div class="loader" />
            {:else}
                <S3Fields
                    toggleLabel="使用 S3 协议"
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
                                    如果已经存在了附件 想要同步
                                    <strong>
                                        {originalFormSettings.s3?.enabled
                                            ? "S3 协议"
                                            : "本地目录"}
                                    </strong>
                                    to the
                                    <strong
                                        >{formSettings.s3.enabled
                                            ? "S3 协议"
                                            : "本地目录"}</strong
                                    >.
                                    <br />
                                    可以使用以下工具:
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
                                <span class="txt">无法链接s3</span>
                            </div>
                        {:else}
                            <div class="label label-sm label-success entrance-right">
                                <i class="ri-checkbox-circle-line txt-success" />
                                <span class="txt">S3链接成功</span>
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
                            <span class="txt">重置</span>
                        </button>
                    {/if}

                    <button
                        type="submit"
                        class="btn btn-expanded"
                        class:btn-loading={isSaving}
                        disabled={!hasChanges || isSaving}
                        on:click={() => save()}
                    >
                        <span class="txt">保存</span>
                    </button>
                </div>
            {/if}
        </form>
    </div>
</PageWrapper>
