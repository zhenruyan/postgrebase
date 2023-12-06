<script>
    import Field from "@/components/base/Field.svelte";
    import OverlayPanel from "@/components/base/OverlayPanel.svelte";
    import { setErrors } from "@/stores/errors";
    import { addInfoToast, addSuccessToast } from "@/stores/toasts";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";
    import { createEventDispatcher, onDestroy } from "svelte";

    const dispatch = createEventDispatcher();

    const formId = "backup_create_" + CommonHelper.randomString(5);

    let panel;
    let name = "";
    let isSubmitting = false;
    let submitTimeoutId;

    export function show(newName) {
        setErrors({});
        isSubmitting = false;
        name = newName || "";
        panel?.show();
    }

    export function hide() {
        return panel?.hide();
    }

    async function submit() {
        if (isSubmitting) {
            return;
        }

        isSubmitting = true;

        clearTimeout(submitTimeoutId);
        submitTimeoutId = setTimeout(() => {
            hide();
        }, 1500);

        try {
            await ApiClient.backups.create(name, { $cancelKey: formId });

            isSubmitting = false;

            hide();
            dispatch("submit");
            addSuccessToast("Successfully generated new backup.");
        } catch (err) {
            if (!err.isAbort) {
                ApiClient.error(err);
            }
        }

        clearTimeout(submitTimeoutId);
        isSubmitting = false;
    }

    onDestroy(() => {
        clearTimeout(submitTimeoutId);
    });
</script>

<OverlayPanel
    bind:this={panel}
    class="backup-create-panel"
    beforeOpen={() => {
        if (isSubmitting) {
            addInfoToast("请稍等，正在进行备份");
            return false;
        }

        return true;
    }}
    beforeHide={() => {
        if (isSubmitting) {
            addInfoToast(
                "备份的定时任务已启动 。",
                4500
            );
        }

        return true;
    }}
    popup
    on:show
    on:hide
>
    <svelte:fragment slot="header">
        <h4 class="center txt-break">初始化一个新备份</h4>
    </svelte:fragment>

    <div class="alert alert-info">
        <div class="icon">
            <i class="ri-information-line" />
        </div>
        <div class="content">
            <p>
               备份会导致性能收到影响。
            </p>
            <p class="txt-bold">
                云端的备份不会被备份到zip中
            </p>
        </div>
    </div>

    <form id={formId} autocomplete="off" on:submit|preventDefault={submit}>
        <Field class="form-field m-0" name="name" let:uniqueId>
            <label for={uniqueId}>备份名</label>
            <input
                type="text"
                id={uniqueId}
                placeholder={"Leave empty to autogenerate"}
                pattern="^[a-z0-9_-]+\.zip$"
                bind:value={name}
            />
            <em class="help-block">格式需要如此 [a-z0-9_-].zip</em>
        </Field>
    </form>

    <svelte:fragment slot="footer">
        <button type="button" class="btn btn-transparent" on:click={hide} disabled={isSubmitting}>
            <span class="txt">取消</span>
        </button>
        <button
            type="submit"
            form={formId}
            class="btn btn-expanded"
            class:btn-loading={isSubmitting}
            disabled={isSubmitting}
        >
            <span class="txt">立即开始</span>
        </button>
    </svelte:fragment>
</OverlayPanel>
