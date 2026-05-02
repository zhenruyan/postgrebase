<script>
    import tooltip from "@/actions/tooltip";
    import Field from "@/components/base/Field.svelte";
    import ObjectSelect from "@/components/base/ObjectSelect.svelte";
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import RedactedPasswordInput from "@/components/base/RedactedPasswordInput.svelte";
    import EmailTemplateAccordion from "@/components/settings/EmailTemplateAccordion.svelte";
    import EmailTestPopup from "@/components/settings/EmailTestPopup.svelte";
    import SettingsSidebar from "@/components/settings/SettingsSidebar.svelte";
    import { t } from "@/i18n";
    import { pageTitle } from "@/stores/app";
    import { setErrors } from "@/stores/errors";
    import { addSuccessToast } from "@/stores/toasts";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";
    import { slide } from "svelte/transition";

    const tlsOptions = [
        { label: "Auto (StartTLS)", value: false },
        { label: "Always", value: true },
    ];

    const authMethods = [
        { label: "PLAIN (default)", value: "PLAIN" },
        { label: "LOGIN", value: "LOGIN" },
    ];

    $: $pageTitle = $t("Mail Settings");

    let testPopup;
    let originalFormSettings = {};
    let formSettings = {};
    let isLoading = false;
    let isSaving = false;

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
            setErrors({});
            addSuccessToast($t("Successfully saved mail settings."));
        } catch (err) {
            ApiClient.error(err);
        }

        isSaving = false;
    }

    function init(settings = {}) {
        formSettings = {
            meta: settings?.meta || {},
            smtp: settings?.smtp || {},
        };

        if (!formSettings.smtp.authMethod) {
            formSettings.smtp.authMethod = authMethods[0].value;
        }

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
            <div class="breadcrumb-item">{$t("Settings")}</div>
            <div class="breadcrumb-item">{$pageTitle}</div>
        </nav>
    </header>

    <div class="wrapper">
        <form class="panel" autocomplete="off" on:submit|preventDefault={() => save()}>
            <div class="content txt-xl m-b-base">
                <p>{$t("Configure common email sending templates.")}</p>
            </div>

            {#if isLoading}
                <div class="loader" />
            {:else}
                <div class="grid m-b-base">
                    <div class="col-lg-6">
                        <Field class="form-field required" name="meta.senderName" let:uniqueId>
                            <label for={uniqueId}>{$t("Sender name")}</label>
                            <input
                                type="text"
                                id={uniqueId}
                                required
                                bind:value={formSettings.meta.senderName}
                            />
                        </Field>
                    </div>

                    <div class="col-lg-6">
                        <Field class="form-field required" name="meta.senderAddress" let:uniqueId>
                            <label for={uniqueId}>{$t("Sender email address")}</label>
                            <input
                                type="email"
                                id={uniqueId}
                                required
                                bind:value={formSettings.meta.senderAddress}
                            />
                        </Field>
                    </div>
                </div>

                <div class="accordions">
                    <EmailTemplateAccordion
                        single
                        key="meta.verificationTemplate"
                        title={$t("Default verification email template")}
                        bind:config={formSettings.meta.verificationTemplate}
                    />

                    <EmailTemplateAccordion
                        single
                        key="meta.resetPasswordTemplate"
                        title={$t("Default password reset email template")}
                        bind:config={formSettings.meta.resetPasswordTemplate}
                    />

                    <EmailTemplateAccordion
                        single
                        key="meta.confirmEmailChangeTemplate"
                        title={$t("Default email change email template")}
                        bind:config={formSettings.meta.confirmEmailChangeTemplate}
                    />
                </div>

                <hr />

                <Field class="form-field form-field-toggle m-b-sm" let:uniqueId>
                    <input type="checkbox" id={uniqueId} required bind:checked={formSettings.smtp.enabled} />
                    <label for={uniqueId}>
                        <span class="txt">{$t("Use SMTP service")} <strong>({$t("recommended")})</strong></span>
                        <i
                            class="ri-information-line link-hint"
                            use:tooltip={{
                                text: $t("Sendmail is used by default, but SMTP is recommended."),
                                position: "top",
                            }}
                        />
                    </label>
                </Field>

                {#if formSettings.smtp.enabled}
                    <div class="grid" transition:slide|local={{ duration: 150 }}>
                        <div class="col-lg-4">
                            <Field class="form-field required" name="smtp.host" let:uniqueId>
                                <label for={uniqueId}>{$t("SMTP host")}</label>
                                <input
                                    type="text"
                                    id={uniqueId}
                                    required
                                    bind:value={formSettings.smtp.host}
                                />
                            </Field>
                        </div>
                        <div class="col-lg-2">
                            <Field class="form-field required" name="smtp.port" let:uniqueId>
                                <label for={uniqueId}>{$t("Port")}</label>
                                <input
                                    type="number"
                                    id={uniqueId}
                                    required
                                    bind:value={formSettings.smtp.port}
                                />
                            </Field>
                        </div>
                        <div class="col-lg-3">
                            <Field class="form-field required" name="smtp.tls" let:uniqueId>
                                <label for={uniqueId}>{$t("TLS")}</label>
                                <ObjectSelect
                                    id={uniqueId}
                                    items={tlsOptions}
                                    bind:keyOfSelected={formSettings.smtp.tls}
                                />
                            </Field>
                        </div>
                        <div class="col-lg-3">
                            <Field class="form-field" name="smtp.authMethod" let:uniqueId>
                                <label for={uniqueId}>{$t("Auth method")}</label>
                                <ObjectSelect
                                    id={uniqueId}
                                    items={authMethods}
                                    bind:keyOfSelected={formSettings.smtp.authMethod}
                                />
                            </Field>
                        </div>
                        <div class="col-lg-6">
                            <Field class="form-field" name="smtp.username" let:uniqueId>
                                <label for={uniqueId}>{$t("SMTP username")}</label>
                                <input type="text" id={uniqueId} bind:value={formSettings.smtp.username} />
                            </Field>
                        </div>
                        <div class="col-lg-6">
                            <Field class="form-field" name="smtp.password" let:uniqueId>
                                <label for={uniqueId}>{$t("SMTP password")}</label>
                                <RedactedPasswordInput
                                    id={uniqueId}
                                    bind:value={formSettings.smtp.password}
                                />
                            </Field>
                        </div>
                        <!-- margin helper -->
                        <div class="col-lg-12" />
                    </div>
                {/if}

                <div class="flex">
                    <div class="flex-fill" />

                    {#if hasChanges}
                        <button
                            type="button"
                            class="btn btn-transparent btn-hint"
                            disabled={isSaving}
                            on:click={() => reset()}
                        >
                            <span class="txt">{$t("Cancel")}</span>
                        </button>
                        <button
                            type="submit"
                            class="btn btn-expanded"
                            class:btn-loading={isSaving}
                            disabled={!hasChanges || isSaving}
                            on:click={() => save()}
                        >
                            <span class="txt">{$t("Save")}</span>
                        </button>
                    {:else}
                        <button
                            type="button"
                            class="btn btn-expanded btn-outline"
                            on:click={() => testPopup?.show()}
                        >
                            <i class="ri-mail-check-line" />
                            <span class="txt">{$t("Send test email")}</span>
                        </button>
                    {/if}
                </div>
            {/if}
        </form>
    </div>
</PageWrapper>

<EmailTestPopup bind:this={testPopup} />
