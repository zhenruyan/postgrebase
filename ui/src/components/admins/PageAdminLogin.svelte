<script>
    import Field from "@/components/base/Field.svelte";
    import FullPage from "@/components/base/FullPage.svelte";
    import { t } from "@/i18n";
    import { addErrorToast, removeAllToasts } from "@/stores/toasts";
    import ApiClient from "@/utils/ApiClient";
    import { link, querystring, replace } from "svelte-spa-router";

    const queryParams = new URLSearchParams($querystring);

    let email = queryParams.get("demoEmail") || "";
    let password = queryParams.get("demoPassword") || "";
    let isLoading = false;

    function login() {
        if (isLoading) {
            return;
        }

        isLoading = true;

        return ApiClient.admins
            .authWithPassword(email, password)
            .then(() => {
                removeAllToasts();
                replace("/");
            })
            .catch(() => {
                addErrorToast("Invalid login credentials.");
            })
            .finally(() => {
                isLoading = false;
            });
    }
</script>

<FullPage>
    <form class="block" on:submit|preventDefault={login}>
        <div class="content txt-center m-b-base">
            <h4>{$t("Login")}</h4>
        </div>

        <Field class="form-field required" name="identity" let:uniqueId>
            <label for={uniqueId}>{$t("Email")}</label>
            <!-- svelte-ignore a11y-autofocus -->
            <input type="email" id={uniqueId} bind:value={email} required autofocus />
        </Field>

        <Field class="form-field required" name="password" let:uniqueId>
            <label for={uniqueId}>{$t("Password")}</label>
            <input type="password" id={uniqueId} bind:value={password} required />
            <div class="help-block">
                <a href="/request-password-reset" class="link-hint" use:link>{$t("Forgot password?")}</a>
            </div>
        </Field>

        <button
            type="submit"
            class="btn btn-lg btn-block btn-next"
            class:btn-disabled={isLoading}
            class:btn-loading={isLoading}
        >
            <span class="txt">{$t("Login")}</span>
            <i class="ri-arrow-right-line" />
        </button>
    </form>
</FullPage>
