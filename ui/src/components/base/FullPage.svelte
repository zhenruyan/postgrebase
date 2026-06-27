<script>
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import { locale } from "@/i18n";

    export let nobranding = false;

    function toggleLocale() {
        locale.set($locale === "cn" ? "en" : "cn");
    }
</script>

<PageWrapper class="full-page" center>
    <div class="wrapper wrapper-sm m-b-xl panel-wrapper">
        <button type="button" class="lang-toggle" on:click={toggleLocale}>
            <i class="ri-translate" />
            <span>{$locale === "cn" ? "English" : "中文"}</span>
        </button>

        {#if !nobranding}
            <div class="block txt-center m-b-lg">
                <figure class="logo">
                    <img
                        src="{import.meta.env.BASE_URL}images/logo.png"
                        alt="PostgresBase logo"
                        width="40"
                        height="40"
                    />
                    <span class="txt">Postgres<strong>Base</strong></span>
                </figure>
            </div>
            <div class="clearfix" />
        {/if}

        <slot />
    </div>
</PageWrapper>

<style>
    .wrapper {
        position: relative;
    }

    .panel-wrapper {
        animation: slideIn 200ms;
    }

    .lang-toggle {
        position: absolute;
        top: 0;
        right: 0;
        display: inline-flex;
        gap: 6px;
        align-items: center;
        padding: 6px 8px;
        color: var(--txtHintColor);
        background: transparent;
        border: 0;
        cursor: pointer;
    }

    .lang-toggle:hover,
    .lang-toggle:focus-visible {
        color: var(--txtPrimaryColor);
    }
</style>
