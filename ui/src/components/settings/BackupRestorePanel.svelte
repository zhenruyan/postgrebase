<script>
    import CopyIcon from "@/components/base/CopyIcon.svelte";
    import Field from "@/components/base/Field.svelte";
    import OverlayPanel from "@/components/base/OverlayPanel.svelte";
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
        <h4 class="center txt-break">恢复 <strong>{name}</strong>的备份</h4>
    </svelte:fragment>

    <div class="alert alert-danger">
        <div class="icon">
            <i class="ri-alert-line" />
        </div>
        <div class="content">
            <p>请谨慎操作</p>
            <p>
               备份会导致pb_data目录被替换。有可能会损失文件。
            </p>
            <p class="txt-bold">
                备份系统还不稳定，更推荐在服务器直接备份。
            </p>
        </div>
    </div>

    <div class="content m-b-sm">
        请输入
        <div class="label">
            <span class="txt">{name}</span>
            <CopyIcon value={name} />
        </div>
        备份的全名:
    </div>

    <form id={formId} autocomplete="off" on:submit|preventDefault={submit}>
        <Field class="form-field required m-0" name="name" let:uniqueId>
            <label for={uniqueId}>备份名</label>
            <input type="text" id={uniqueId} required bind:value={nameConfirm} />
        </Field>
    </form>

    <svelte:fragment slot="footer">
        <button type="button" class="btn btn-transparent" on:click={hide} disabled={isSubmitting}>
            取消
        </button>
        <button
            type="submit"
            form={formId}
            class="btn btn-expanded"
            class:btn-loading={isSubmitting}
            disabled={!canSubmit || isSubmitting}
        >
            <span class="txt">开始恢复</span>
        </button>
    </svelte:fragment>
</OverlayPanel>
