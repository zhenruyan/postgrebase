<script>
    import Field from "@/components/base/Field.svelte";
    import { t } from "@/i18n";

    export let collection;
</script>

<div class="grid">
    <div class="col-lg-12">
        <div class="section-title">{$t("Redis Cache Settings")}</div>
    </div>

    <div class="col-lg-4">
        <Field class="form-field" name="cacheEnabled" let:uniqueId>
            <div class="form-field-checkbox">
                <input type="checkbox" id={uniqueId} bind:checked={collection.cacheEnabled} />
                <label for={uniqueId}>{$t("Enable Single Object Cache")}</label>
            </div>
            <div class="help-block">{$t("Cache individual records by their ID.")}</div>
        </Field>
    </div>

    <div class="col-lg-4">
        <Field class="form-field" name="listCacheEnabled" let:uniqueId>
            <div class="form-field-checkbox">
                <input type="checkbox" id={uniqueId} bind:checked={collection.listCacheEnabled} />
                <label for={uniqueId}>{$t("Enable List Cache")}</label>
            </div>
            <div class="help-block">{$t("Cache full collection lists.")}</div>
        </Field>
    </div>

    <div class="col-lg-4">
        <Field class="form-field" name="searchCacheEnabled" let:uniqueId>
            <div class="form-field-checkbox">
                <input type="checkbox" id={uniqueId} bind:checked={collection.searchCacheEnabled} />
                <label for={uniqueId}>{$t("Enable Search/Query Cache")}</label>
            </div>
            <div class="help-block">{$t("Cache filtered search results.")}</div>
        </Field>
    </div>

    <div class="col-lg-6">
        <Field class="form-field" name="cacheDuration" let:uniqueId>
            <label for={uniqueId}>{$t("Cache Duration (seconds)")}</label>
            <input type="number" id={uniqueId} bind:value={collection.cacheDuration} min="0" step="1" />
            <div class="help-block">{$t("How long the data should stay in Redis (0 = infinite).")}</div>
        </Field>
    </div>
</div>

<style>
    .section-title {
        margin-bottom: var(--smSpacing);
        font-weight: bold;
        color: var(--txtHintColor);
        text-transform: uppercase;
        font-size: 0.85rem;
    }
</style>
