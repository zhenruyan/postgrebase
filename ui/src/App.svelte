<script>
    import "./scss/main.scss";

    import tooltip from "@/actions/tooltip";
    import Confirmation from "@/components/base/Confirmation.svelte";
    import Toasts from "@/components/base/Toasts.svelte";
    import Toggler from "@/components/base/Toggler.svelte";
    import { admin } from "@/stores/admin";
    import { appName, hideControls, pageTitle } from "@/stores/app";
    import { resetConfirmation } from "@/stores/confirmation";
    import { setErrors } from "@/stores/errors";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";
    import TinyMCE from "@tinymce/tinymce-svelte";
    import Router, { link, replace } from "svelte-spa-router";
    import active from "svelte-spa-router/active";
    import routes from "./routes";

    let oldLocation = undefined;

    let showAppSidebar = false;

    let isTinyMCEPreloaded = false;

    $: if ($admin?.id) {
        loadSettings();
    }

    function handleRouteLoading(e) {
        if (e?.detail?.location === oldLocation) {
            return; // not an actual change
        }

        showAppSidebar = !!e?.detail?.userData?.showAppSidebar;

        oldLocation = e?.detail?.location;

        // resets
        $pageTitle = "";
        setErrors({});
        resetConfirmation();
    }

    function handleRouteFailure() {
        replace("/");
    }

    async function loadSettings() {
        if (!$admin?.id) {
            return;
        }

        try {
            const settings = await ApiClient.settings.getAll({
                $cancelKey: "initialAppSettings",
            });
            $appName = settings?.meta?.appName || "";
            $hideControls = !!settings?.meta?.hideControls;
        } catch (err) {
            if (!err?.isAbort) {
                console.warn("Failed to load app settings.", err);
            }
        }
    }

    function logout() {
        ApiClient.logout();
    }
</script>

<svelte:head>
    <title>{CommonHelper.joinNonEmpty([$pageTitle, $appName, "PostgresBase"], " - ")}</title>
</svelte:head>

<div class="app-layout">
    {#if $admin?.id && showAppSidebar}
        <aside class="app-sidebar">
            <a href="/" class="logo logo-sm" use:link>
                <img
                    src="{import.meta.env.BASE_URL}images/logo.png"
                    alt="PostgresBase logo"
                    width="40"
                    height="40"
                />
            </a>

            <nav class="main-menu">
                <a
                    href="/collections"
                    class="menu-item"
                    aria-label="Collections"
                    use:link
                    use:active={{ path: "/collections/?.*", className: "current-route" }}
                    use:tooltip={{ text: "表结构", position: "right" }}
                >
                    <div>
                        <div>
                            <i class="ri-database-2-line" />
                        </div>
                        <p>表结构</p>
                    </div>
                </a>

                <a
                    href="/settings"
                    class="menu-item"
                    aria-label="Settings"
                    use:link
                    use:active={{ path: "/settings/?.*", className: "current-route" }}
                    use:tooltip={{ text: "设置", position: "right" }}
                >
                    <div>
                        <div><i class="ri-tools-line" /></div>
                        <p>设置</p>
                    </div>
                </a>
            </nav>

            <figure class="thumb thumb-circle link-hint closable">
                <img
                    src="{import.meta.env.BASE_URL}images/avatars/avatar{$admin?.avatar || 0}.svg"
                    alt="Avatar"
                />
                <Toggler class="dropdown dropdown-nowrap dropdown-upside dropdown-left">
                    <a href="/settings/admins" class="dropdown-item closable" use:link>
                        <i class="ri-shield-user-line" />
                        <span class="txt">管理用户</span>
                    </a>
                    <hr />
                    <button type="button" class="dropdown-item closable" on:click={logout}>
                        <i class="ri-logout-circle-line" />
                        <span class="txt">退出登录</span>
                    </button>
                </Toggler>
            </figure>
        </aside>
    {/if}

    <div class="app-body">
        <Router {routes} on:routeLoading={handleRouteLoading} on:conditionsFailed={handleRouteFailure} />

        <Toasts />
    </div>
</div>

<Confirmation />

{#if showAppSidebar && !isTinyMCEPreloaded}
    <div class="tinymce-preloader hidden">
        <TinyMCE
            scriptSrc="{import.meta.env.BASE_URL}libs/tinymce/tinymce.min.js"
            conf={CommonHelper.defaultEditorOptions()}
            on:init={() => {
                isTinyMCEPreloaded = true;
            }}
        />
    </div>
{/if}
