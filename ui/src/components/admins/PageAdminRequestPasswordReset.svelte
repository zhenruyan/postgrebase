<script>
    import Field from "@/components/base/Field.svelte";
    import FullPage from "@/components/base/FullPage.svelte";
    import { t } from "@/i18n";
    import ApiClient from "@/utils/ApiClient";
    import { link } from "svelte-spa-router";

    let email = "";
    let isLoading = false;
    let success = false;

    async function submit() {
        if (isLoading) {
            return;
        }

        isLoading = true;

        try {
            await ApiClient.admins.requestPasswordReset(email);
            success = true;
        } catch (err) {
            ApiClient.error(err);
        }

        isLoading = false;
    }
</script>

<FullPage>
    {#if success}
        <div class="alert alert-success">
            <div class="icon"><i class="ri-checkbox-circle-line" /></div>
            <div class="content">
                <p>
                    {$t("Check {email} for password reset instructions.", { email })}
                </p>
            </div>
        </div>
    {:else}
        <form class="m-b-base" on:submit|preventDefault={submit}>
            <div class="content txt-center m-b-sm">
                <h4 class="m-b-xs">{$t("Forgot password")}</h4>
                <p>{$t("Send password reset instructions to your email.")}</p>
            </div>

            <Field class="form-field required" name="email" let:uniqueId>
                <label for={uniqueId}>{$t("Email")}</label>
                <!-- svelte-ignore a11y-autofocus -->
                <input type="email" id={uniqueId} required autofocus bind:value={email} />
            </Field>

            <button
                type="submit"
                class="btn btn-lg btn-block"
                class:btn-loading={isLoading}
                disabled={isLoading}
            >
                <i class="ri-mail-send-line" />
                <span class="txt">{$t("Send reset email")}</span>
            </button>
        </form>
    {/if}

    <div class="content txt-center">
        <a href="/login" class="link-hint" use:link>{$t("Back to login")}</a>
    </div>
</FullPage>
