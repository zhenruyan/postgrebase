<script>
    import { createEventDispatcher } from "svelte";
    import { setErrors } from "@/stores/errors";
    import { addSuccessToast } from "@/stores/toasts";
    import { confirm } from "@/stores/confirmation";
    import { t } from "@/i18n";
    import ApiClient from "@/utils/ApiClient";
    import OverlayPanel from "@/components/base/OverlayPanel.svelte";
    import Field from "@/components/base/Field.svelte";

    const dispatch = createEventDispatcher();

    let panel;
    let model = {};
    let isSaving = false;

    export function show(res) {
        setErrors({});
        model = res ? { ...res } : { name: "" };
        panel?.show();
    }

    export function hide() {
        panel?.hide();
    }

    async function save() {
        if (isSaving) return;
        isSaving = true;

        try {
            let res;
            if (model.id) {
                res = await ApiClient.collection("project").update(model.id, model);
            } else {
                res = await ApiClient.collection("project").create(model);
            }
            addSuccessToast(model.id ? $t("Save") : $t("Create"));
            dispatch("save", res);
            hide();
        } catch (err) {
            ApiClient.error(err);
        } finally {
            isSaving = false;
        }
    }

    function remove() {
        confirm($t("Are you sure you want to delete project \"{name}\"?", { name: model.name }), async () => {
            try {
                await ApiClient.collection("project").delete(model.id);
                addSuccessToast($t("Delete"));
                dispatch("delete", model);
                hide();
            } catch (err) {
                ApiClient.error(err);
            }
        });
    }
</script>

<OverlayPanel bind:this={panel} class="overlay-panel-sm" on:hide on:show>
    <svelte:fragment slot="header">
        <h4>{model.id ? $t("Edit Project") : $t("New Project")}</h4>
    </svelte:fragment>

    <form on:submit|preventDefault={save}>
        <Field class="form-field required" name="name" let:uniqueId>
            <label for={uniqueId}>{$t("Project Name")}</label>
            <input type="text" id={uniqueId} bind:value={model.name} required />
        </Field>

        <input type="submit" class="hidden" />
    </form>

    <svelte:fragment slot="footer">
        <button type="button" class="btn btn-transparent" on:click={hide}>{$t("Cancel")}</button>
        {#if model.id}
            <button type="button" class="btn btn-transparent txt-danger" on:click={remove}>{$t("Delete")}</button>
        {/if}
        <button
            type="button"
            class="btn btn-expanded"
            class:btn-loading={isSaving}
            disabled={isSaving}
            on:click={save}
        >
            <span class="txt">{model.id ? $t("Save") : $t("Create")}</span>
        </button>
    </svelte:fragment>
</OverlayPanel>
