<script>
    import { SchemaField } from "pocketbase";
    import Field from "@/components/base/Field.svelte";

    export let field = new SchemaField();
    export let value = false;
</script>

<Field class="form-field form-field-toggle {field.required ? 'required' : ''}" name={field.name} let:uniqueId>
    <input type="checkbox" id={uniqueId} bind:checked={value} />
    <label for={uniqueId}>
        {field.name}
        {#if field.remark}
            <span class="txt-hint">({field.remark})</span>
        {/if}
    </label>
</Field>
