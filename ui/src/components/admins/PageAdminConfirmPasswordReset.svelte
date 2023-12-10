<script>
    import Field from "@/components/base/Field.svelte";
    import FullPage from "@/components/base/FullPage.svelte";
    import { addSuccessToast } from "@/stores/toasts";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";
    import { link, replace } from "svelte-spa-router";

    export let params;

    let newPassword = "";
    let newPasswordConfirm = "";
    let isLoading = false;

    $: email = CommonHelper.getJWTPayload(params?.token).email || "";

    async function submit() {
        if (isLoading) {
            return;
        }

        isLoading = true;

        try {
            await ApiClient.admins.confirmPasswordReset(params?.token, newPassword, newPasswordConfirm);
            addSuccessToast("成功设置新的密码");
            replace("/");
        } catch (err) {
            ApiClient.error(err);
        }

        isLoading = false;
    }
</script>

<FullPage>
    <form class="m-b-base" on:submit|preventDefault={submit}>
        <div class="content txt-center m-b-sm">
            <h4 class="m-b-xs">
                重置管理员密码
                {#if email}
                    for <strong class="txt-nowrap">{email}</strong>
                {/if}
            </h4>
        </div>

        <Field class="form-field required" name="password" let:uniqueId>
            <label for={uniqueId}>新密码</label>
            <!-- svelte-ignore a11y-autofocus -->
            <input type="password" id={uniqueId} required autofocus bind:value={newPassword} />
        </Field>

        <Field class="form-field required" name="passwordConfirm" let:uniqueId>
            <label for={uniqueId}>验证新密码</label>
            <input type="password" id={uniqueId} required bind:value={newPasswordConfirm} />
        </Field>

        <button type="submit" class="btn btn-lg btn-block" class:btn-loading={isLoading} disabled={isLoading}>
            <span class="txt">设置新密码</span>
        </button>
    </form>

    <div class="content txt-center">
        <a href="/login" class="link-hint" use:link>返回到登录页面</a>
    </div>
</FullPage>
