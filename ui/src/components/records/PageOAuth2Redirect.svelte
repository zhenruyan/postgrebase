<script>
    import { t } from "@/i18n";
    import { onMount } from "svelte";

    onMount(() => {
        window.close();
    });
</script>

<div class="content txt-hint txt-center p-base">
    <h3 class="m-b-sm">{$t("OAuth login")}</h3>
    <h5>{$t("If this window doesn't close automatically, close it manually.")}</h5>
</div>
