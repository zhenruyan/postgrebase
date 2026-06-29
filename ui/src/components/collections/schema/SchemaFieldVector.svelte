<script>
    import SchemaField from "@/components/collections/schema/SchemaField.svelte";
    import Field from "@/components/base/Field.svelte";
    import { t } from "@/i18n";

    export let field;
    export let key; // the index in the schema array
    export let collection;

    // Filter fields from same collection to bind to. Only non-vector fields!
    $: availableSourceFields = (collection?.schema || []).filter(
        (f) => f.name && f.name !== field.name && f.type !== "vector" && !f.toDelete
    );

    // Initialize options
    $: field.options = field.options || {};
</script>

<SchemaField bind:field {key} on:rename on:remove {...$$restProps}>
    <div slot="options" let:interactive>
        <div class="grid grid-sm">
            <div class="col-sm-12">
                <Field class="form-field required {!interactive ? 'disabled' : ''}" name="schema.{key}.options.sourceField" let:uniqueId>
                    <label for={uniqueId}>{$t("Source Text Field (to generate embeddings from)")}</label>
                    <select
                        id={uniqueId}
                        disabled={!interactive}
                        bind:value={field.options.sourceField}
                    >
                        <option value="">-- Select text/editor field --</option>
                        {#each availableSourceFields as f}
                            <option value={f.name}>{f.name} ({f.type})</option>
                        {/each}
                    </select>
                </Field>
            </div>
        </div>
    </div>
</SchemaField>