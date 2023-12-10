<script>
    import { link } from "svelte-spa-router";
    import active from "svelte-spa-router/active";
</script>

<aside class="page-sidebar settings-sidebar">
    <div class="sidebar-content">
        <div class="sidebar-title">系统管理</div>
        <a href="/settings" class="sidebar-list-item" use:active={{ path: "/settings" }} use:link>
            <i class="ri-home-gear-line" />
            <span class="txt">应用设置</span>
        </a>
        <a
            href="/settings/mail"
            class="sidebar-list-item"
            use:active={{ path: "/settings/mail/?.*" }}
            use:link
        >
            <i class="ri-send-plane-2-line" />
            <span class="txt">邮箱设置</span>
        </a>
        <a
            href="/settings/storage"
            class="sidebar-list-item"
            use:active={{ path: "/settings/storage/?.*" }}
            use:link
        >
            <i class="ri-archive-drawer-line" />
            <span class="txt">文件存储</span>
        </a>
        <a
            href="/settings/backups"
            class="sidebar-list-item"
            use:active={{ path: "/settings/backups/?.*" }}
            use:link
        >
            <i class="ri-archive-line" />
            <span class="txt">附件备份</span>
        </a>

        <div class="sidebar-title">
            <span class="txt">迁移</span>
        </div>
        <a
            href="/settings/export-collections"
            class="sidebar-list-item"
            use:active={{ path: "/settings/export-collections/?.*" }}
            use:link
        >
            <i class="ri-uninstall-line" />
            <span class="txt">导出表结构</span>
        </a>
        <a
            href="/settings/import-collections"
            class="sidebar-list-item"
            use:active={{ path: "/settings/import-collections/?.*" }}
            use:link
        >
            <i class="ri-install-line" />
            <span class="txt">导入表结构</span>
        </a>

        <div class="sidebar-title">Auth设置</div>
        <a
            href="/settings/auth-providers"
            class="sidebar-list-item"
            use:active={{ path: "/settings/auth-providers/?.*" }}
            use:link
        >
            <i class="ri-lock-password-line" />
            <span class="txt">第三方登录设置</span>
        </a>
        <a
            href="/settings/tokens"
            class="sidebar-list-item"
            use:active={{ path: "/settings/tokens/?.*" }}
            use:link
        >
            <i class="ri-key-line" />
            <span class="txt">Token 设置</span>
        </a>
        <a
            href="/settings/admins"
            class="sidebar-list-item"
            use:active={{ path: "/settings/admins/?.*" }}
            use:link
        >
            <i class="ri-shield-user-line" />
            <span class="txt">管理员管理</span>
        </a>
    </div>
</aside>
