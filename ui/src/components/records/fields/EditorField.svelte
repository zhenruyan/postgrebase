<script>
    import { SchemaField } from "pocketbase";
    import CommonHelper from "@/utils/CommonHelper";
    import Field from "@/components/base/Field.svelte";
    import TinyMCE from "@tinymce/tinymce-svelte";

    export let field = new SchemaField();
    export let value = undefined;
</script>

<Field class="form-field {field.required ? 'required' : ''}" name={field.name} let:uniqueId>
    <label for={uniqueId}>
        <i class={CommonHelper.getFieldTypeIcon(field.type)} />
        <span class="txt">
            {field.name}
            {#if field.remark}
                <span class="txt-hint">({field.remark})</span>
            {/if}
        </span>
    </label>
    <TinyMCE
        id={uniqueId}
        scriptSrc="{import.meta.env.BASE_URL}libs/tinymce/tinymce.min.js"
        conf={CommonHelper.defaultEditorOptions()}
        bind:value
    />
</Field>
