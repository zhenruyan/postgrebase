<script>
    import { onMount } from "svelte";
    import { slide } from "svelte/transition";
    import ApiClient from "@/utils/ApiClient";
    import { removeError } from "@/stores/errors";
    import Field from "@/components/base/Field.svelte";
    import RedactedPasswordInput from "@/components/base/RedactedPasswordInput.svelte";

    const testRequestKey = "webdav_test_request";

    export let originalConfig = {};
    export let config = {};
    export let configKey = "webdav";
    export let toggleLabel = "Enable WebDAV";
    export let testFilesystem = "storage"; // storage or backups
    export let testError = null;
    export let isTesting = false;

    let testTimeoutId = null;
    let testDebounceId = null;

    $: if (originalConfig?.enabled) {
        testConnectionWithDebounce(100);
    }

    // clear webdav errors on disable
    $: if (!config.enabled) {
        removeError(configKey);
    }

    function testConnectionWithDebounce(timeout) {
        isTesting = true;
        clearTimeout(testDebounceId);
        testDebounceId = setTimeout(() => {
            testConnection();
        }, timeout);
    }

    async function testConnection() {
        testError = null;

        if (!config.enabled) {
            isTesting = false;
            return testError; // nothing to test
        }

        // auto cancel the test request after 30sec
        ApiClient.cancelRequest(testRequestKey);
        clearTimeout(testTimeoutId);
        testTimeoutId = setTimeout(() => {
            ApiClient.cancelRequest(testRequestKey);
            testError = new Error("WebDAV test connection timeout.");
            isTesting = false;
        }, 30000);

        isTesting = true;

        let err;

        try {
            await ApiClient.settings.testWebDAV(testFilesystem, {
                $cancelKey: testRequestKey,
            });
        } catch (e) {
            err = e;
        }

        if (!err?.isAbort) {
            testError = err;
            isTesting = false;
            clearTimeout(testTimeoutId);
        }

        return testError;
    }

    onMount(() => {
        return () => {
            clearTimeout(testTimeoutId);
            clearTimeout(testDebounceId);
        };
    });
</script>

<Field class="form-field form-field-toggle" let:uniqueId>
    <input type="checkbox" id={uniqueId} required bind:checked={config.enabled} />
    <label for={uniqueId}>{toggleLabel}</label>
</Field>

<slot {isTesting} {testError} enabled={config.enabled} />

{#if config.enabled}
    <div class="grid" transition:slide|local={{ duration: 150 }}>
        <div class="col-lg-12">
            <Field class="form-field required" name="{configKey}.url" let:uniqueId>
                <label for={uniqueId}>WebDAV URL</label>
                <input type="text" id={uniqueId} required bind:value={config.url} />
            </Field>
        </div>
        <div class="col-lg-6">
            <Field class="form-field required" name="{configKey}.username" let:uniqueId>
                <label for={uniqueId}>Username</label>
                <input type="text" id={uniqueId} required bind:value={config.username} />
            </Field>
        </div>
        <div class="col-lg-6">
            <Field class="form-field required" name="{configKey}.password" let:uniqueId>
                <label for={uniqueId}>Password</label>
                <RedactedPasswordInput id={uniqueId} required bind:value={config.password} />
            </Field>
        </div>
        <!-- margin helper -->
        <div class="col-lg-12" />
    </div>
{/if}
