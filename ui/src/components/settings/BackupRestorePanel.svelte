<script>
    import CopyIcon from "@/components/base/CopyIcon.svelte";
    import Field from "@/components/base/Field.svelte";
    import OverlayPanel from "@/components/base/OverlayPanel.svelte";
    import { t } from "@/i18n";
    import { setErrors } from "@/stores/errors";
    import { addErrorToast } from "@/stores/toasts";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";

    const formId = "backup_restore_" + CommonHelper.randomString(5);

    let panel;
    let name = "";
    let nameConfirm = "";
    let isSubmitting = false;

    $: canSubmit = nameConfirm != "" && name == nameConfirm;

    export function show(backupName) {
        setErrors({});
        nameConfirm = "";
        name = backupName;
        isSubmitting = false;
        panel?.show();
    }

    export function hide() {
        return panel?.hide();
    }

    async function submit() {
        if (!canSubmit || isSubmitting) {
            return;
        }

        isSubmitting = true;

        try {
            await ApiClient.backups.restore(name);

            // slight delay just in case the application is still restarting
            setTimeout(() => {
                window.location.reload();
            }, 1000);
        } catch (err) {
            if (!err?.isAbort) {
                isSubmitting = false;
                addErrorToast(err.response?.message || err.message);
            }
        }
    }
</script>

<OverlayPanel
    bind:this={panel}
    class="backup-restore-panel"
    overlayClose={!isSubmitting}
    escClose={!isSubmitting}
    beforeHide={() => !isSubmitting}
    popup
    on:show
    on:hide
>
    <svelte:fragment slot="header">
        <h4 class="center txt-break">{$t("Restore backup")} <strong>{name}</strong></h4>
    </svelte:fragment>

    <div class="alert alert-danger">
        <div class="icon">
            <i class="ri-alert-line" />
        </div>
        <div class="content">
            <p>{$t("Proceed carefully.")}</p>
            <p>
                {$t("Restoring a backup replaces the pb_data directory and may cause file loss.")}
            </p>
            <p class="txt-bold">
                {$t("Server-level backups are recommended for production.")}
            </p>
        </div>
    </div>

    <div class="content m-b-sm">
        {$t("Enter")}
        <div class="label">
            <span class="txt">{name}</span>
            <CopyIcon value={name} />
        </div>
        {$t("to confirm the restore.")}
    </div>

    <form id={formId} autocomplete="off" on:submit|preventDefault={submit}>
        <Field class="form-field required m-0" name="name" let:uniqueId>
            <label for={uniqueId}>{$t("Backup name")}</label>
            <input type="text" id={uniqueId} required bind:value={nameConfirm} />
        </Field>
    </form>

    <svelte:fragment slot="footer">
        <button type="button" class="btn btn-transparent" on:click={hide} disabled={isSubmitting}>
            {$t("Cancel")}
        </button>
        <button
            type="submit"
            form={formId}
            class="btn btn-expanded"
            class:btn-loading={isSubmitting}
            disabled={!canSubmit || isSubmitting}
        >
            <span class="txt">{$t("Start restore")}</span>
        </button>
    </svelte:fragment>
</OverlayPanel>
