<script>
    import { link } from "svelte-spa-router";
    import active from "svelte-spa-router/active";
    import { t } from "@/i18n";
</script>

<aside class="page-sidebar settings-sidebar">
    <div class="sidebar-content">
        <div class="sidebar-title">{$t("System Management")}</div>
        <a href="/settings" class="sidebar-list-item" use:active={{ path: "/settings" }} use:link>
            <i class="ri-home-gear-line" />
            <span class="txt">{$t("Application")}</span>
        </a>
        <a
            href="/settings/mail"
            class="sidebar-list-item"
            use:active={{ path: "/settings/mail/?.*" }}
            use:link
        >
            <i class="ri-send-plane-2-line" />
            <span class="txt">{$t("Mail Settings")}</span>
        </a>
        <a
            href="/settings/storage"
            class="sidebar-list-item"
            use:active={{ path: "/settings/storage/?.*" }}
            use:link
        >
            <i class="ri-archive-drawer-line" />
            <span class="txt">{$t("File Storage")}</span>
        </a>
        <a
            href="/settings/backups"
            class="sidebar-list-item"
            use:active={{ path: "/settings/backups/?.*" }}
            use:link
        >
            <i class="ri-archive-line" />
            <span class="txt">{$t("Backups")}</span>
        </a>

        <div class="sidebar-title">
            <span class="txt">{$t("Migration")}</span>
        </div>
        <a
            href="/settings/export-collections"
            class="sidebar-list-item"
            use:active={{ path: "/settings/export-collections/?.*" }}
            use:link
        >
            <i class="ri-uninstall-line" />
            <span class="txt">{$t("Export Collections")}</span>
        </a>
        <a
            href="/settings/import-collections"
            class="sidebar-list-item"
            use:active={{ path: "/settings/import-collections/?.*" }}
            use:link
        >
            <i class="ri-install-line" />
            <span class="txt">{$t("Import Collections")}</span>
        </a>

        <div class="sidebar-title">{$t("Auth Settings")}</div>
        <a
            href="/settings/auth-providers"
            class="sidebar-list-item"
            use:active={{ path: "/settings/auth-providers/?.*" }}
            use:link
        >
            <i class="ri-lock-password-line" />
            <span class="txt">{$t("Auth Providers")}</span>
        </a>
        <a
            href="/settings/tokens"
            class="sidebar-list-item"
            use:active={{ path: "/settings/tokens/?.*" }}
            use:link
        >
            <i class="ri-key-line" />
            <span class="txt">{$t("Token Options")}</span>
        </a>
        <a
            href="/settings/admins"
            class="sidebar-list-item"
            use:active={{ path: "/settings/admins/?.*" }}
            use:link
        >
            <i class="ri-shield-user-line" />
            <span class="txt">{$t("Admin Management")}</span>
        </a>
    </div>
</aside>
