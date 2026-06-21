<script>
    import CopyIcon from "@/components/base/CopyIcon.svelte";
    import FormattedDate from "@/components/base/FormattedDate.svelte";
    import HorizontalScroller from "@/components/base/HorizontalScroller.svelte";
    import PageWrapper from "@/components/base/PageWrapper.svelte";
    import RefreshButton from "@/components/base/RefreshButton.svelte";
    import SettingsSidebar from "@/components/settings/SettingsSidebar.svelte";
    import { pageTitle } from "@/stores/app";
    import { addSuccessToast, addErrorToast } from "@/stores/toasts";
    import ApiClient from "@/utils/ApiClient";

    $pageTitle = "MCP Tokens";

    let tokens = [];
    let isLoading = false;
    let isCreating = false;
    let showCreateForm = false;
    let newTokenValue = null; // Store the full token value when first created

    // Form fields
    let formName = "";
    let formDescription = "";
    let formExpiresDays = 0;

    // Load tokens on mount
    loadTokens();

    function loadTokens() {
        isLoading = true;
        tokens = [];

        return ApiClient.send("/api/mcp-tokens", {
            method: "GET",
        })
            .then((result) => {
                tokens = result || [];
                isLoading = false;
            })
            .catch((err) => {
                if (!err?.isAbort) {
                    isLoading = false;
                    console.warn(err);
                    ApiClient.error(err, false);
                }
            });
    }

    function showCreate() {
        formName = "";
        formDescription = "";
        formExpiresDays = 0;
        newTokenValue = null;
        showCreateForm = true;
    }

    function hideCreate() {
        showCreateForm = false;
        newTokenValue = null;
    }

    async function createToken() {
        if (!formName.trim()) {
            addErrorToast("请输入 Token 名称");
            return;
        }

        isCreating = true;

        try {
            const result = await ApiClient.send("/api/mcp-tokens", {
                method: "POST",
                body: JSON.stringify({
                    name: formName.trim(),
                    description: formDescription.trim(),
                    expiresDays: formExpiresDays,
                }),
            });

            newTokenValue = result.token;
            addSuccessToast("MCP Token 创建成功！请立即复制保存，此 Token 只会显示一次。");
            loadTokens();
        } catch (err) {
            ApiClient.error(err);
        } finally {
            isCreating = false;
        }
    }

    async function deleteToken(id) {
        if (!confirm("确定要删除此 Token 吗？此操作不可撤销。")) {
            return;
        }

        try {
            await ApiClient.send(`/api/mcp-tokens/${id}`, {
                method: "DELETE",
            });
            addSuccessToast("MCP Token 已删除");
            loadTokens();
        } catch (err) {
            ApiClient.error(err);
        }
    }

    function copyToken(token) {
        navigator.clipboard.writeText(token).then(() => {
            addSuccessToast("已复制到剪贴板");
        }).catch(() => {
            addErrorToast("复制失败");
        });
    }
</script>

<SettingsSidebar />

<PageWrapper>
    <header class="page-header">
        <nav class="breadcrumbs">
            <div class="breadcrumb-item">设置</div>
            <div class="breadcrumb-item">{$pageTitle}</div>
        </nav>

        <RefreshButton on:refresh={() => loadTokens()} />

        <div class="flex-fill" />

        <button type="button" class="btn btn-expanded" on:click={showCreate}>
            <i class="ri-add-line" />
            <span class="txt">新建 Token</span>
        </button>
    </header>

    <!-- Create Token Dialog -->
    {#if showCreateForm}
        <div class="panel panel-highlight">
            <div class="panel-content">
                <h4>创建新的 MCP Token</h4>

                {#if newTokenValue}
                    <div class="alert alert-success m-t-base">
                        <i class="ri-check-line" />
                        <div>
                            <strong>Token 创建成功！</strong>
                            <p class="m-t-xs">请立即复制并保存此 Token，它只会显示一次：</p>
                            <div class="token-display m-t-sm">
                                <code class="token-value">{newTokenValue}</code>
                                <button type="button" class="btn btn-sm btn-success" on:click={() => copyToken(newTokenValue)}>
                                    <i class="ri-file-copy-line" />
                                    复制
                                </button>
                            </div>
                        </div>
                    </div>
                    <div class="m-t-base">
                        <button type="button" class="btn btn-secondary" on:click={hideCreate}>
                            关闭
                        </button>
                    </div>
                {:else}
                    <form on:submit|preventDefault={createToken}>
                        <div class="form-field">
                            <label for="token-name">Token 名称 *</label>
                            <input
                                type="text"
                                id="token-name"
                                class="form-control"
                                placeholder="例如：Claude Desktop, Cursor IDE"
                                bind:value={formName}
                                required
                            />
                        </div>

                        <div class="form-field m-t-sm">
                            <label for="token-description">描述（可选）</label>
                            <textarea
                                id="token-description"
                                class="form-control"
                                rows="2"
                                placeholder="描述此 Token 的用途..."
                                bind:value={formDescription}
                            />
                        </div>

                        <div class="form-field m-t-sm">
                            <label for="token-expires">有效期</label>
                            <select id="token-expires" class="form-control" bind:value={formExpiresDays}>
                                <option value={0}>永不过期</option>
                                <option value={7}>7 天</option>
                                <option value={30}>30 天</option>
                                <option value={90}>90 天</option>
                                <option value={365}>1 年</option>
                            </select>
                        </div>

                        <div class="form-field m-t-base">
                            <button type="submit" class="btn btn-primary" disabled={isCreating}>
                                {#if isCreating}
                                    <i class="ri-loader-line" />
                                    创建中...
                                {:else}
                                    <i class="ri-key-line" />
                                    生成 Token
                                {/if}
                            </button>
                            <button type="button" class="btn btn-secondary m-l-sm" on:click={hideCreate} disabled={isCreating}>
                                取消
                            </button>
                        </div>
                    </form>
                {/if}
            </div>
        </div>
    {/if}

    <!-- Token List -->
    <HorizontalScroller class="table-wrapper">
        <table class="table" class:table-loading={isLoading}>
            <thead>
                <tr>
                    <th>名称</th>
                    <th>Token</th>
                    <th>描述</th>
                    <th>状态</th>
                    <th>过期时间</th>
                    <th>创建时间</th>
                    <th class="min-width" />
                </tr>
            </thead>
            <tbody>
                {#each tokens as token (token.id)}
                    <tr>
                        <td>
                            <strong>{token.name}</strong>
                        </td>
                        <td>
                            <code class="token-masked">{token.token}</code>
                        </td>
                        <td>
                            <span class="txt txt-hint">{token.description || '-'}</span>
                        </td>
                        <td>
                            {#if token.active}
                                <span class="label label-success">启用</span>
                            {:else}
                                <span class="label label-warning">禁用</span>
                            {/if}
                        </td>
                        <td>
                            {#if token.expiresAt}
                                <FormattedDate date={token.expiresAt} />
                            {:else}
                                <span class="txt txt-hint">永不过期</span>
                            {/if}
                        </td>
                        <td>
                            <FormattedDate date={token.created} />
                        </td>
                        <td>
                            <button
                                type="button"
                                class="btn btn-sm btn-hint btn-border"
                                on:click={() => deleteToken(token.id)}
                                title="删除"
                            >
                                <i class="ri-delete-bin-line" />
                            </button>
                        </td>
                    </tr>
                {:else}
                    {#if isLoading}
                        <tr>
                            <td colspan="99" class="p-xs">
                                <span class="skeleton-loader m-0" />
                            </td>
                        </tr>
                    {:else}
                        <tr>
                            <td colspan="99" class="txt-center txt-hint p-xl">
                                <i class="ri-key-line" style="font-size: 2rem; opacity: 0.3;" />
                                <h6 class="m-t-sm">暂无 MCP Token</h6>
                                <p>点击上方"新建 Token"按钮创建一个</p>
                            </td>
                        </tr>
                    {/if}
                {/each}
            </tbody>
        </table>
    </HorizontalScroller>

    {#if tokens.length}
        <small class="block txt-hint txt-right m-t-sm">
            共 {tokens.length} 个 Token
        </small>
    {/if}
</PageWrapper>

<style>
    .token-display {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        flex-wrap: wrap;
    }

    .token-value {
        background: rgba(0, 0, 0, 0.05);
        padding: 0.5rem 1rem;
        border-radius: 4px;
        font-family: monospace;
        font-size: 0.875rem;
        word-break: break-all;
        flex: 1;
        min-width: 200px;
    }

    .token-masked {
        background: rgba(0, 0, 0, 0.05);
        padding: 0.25rem 0.5rem;
        border-radius: 3px;
        font-family: monospace;
        font-size: 0.8125rem;
    }

    .panel-highlight {
        background: var(--baseAlt1Color);
        border: 1px solid var(--primaryColor);
    }

    .alert-success {
        background: rgba(34, 197, 94, 0.1);
        border: 1px solid rgba(34, 197, 94, 0.3);
        border-radius: 4px;
        padding: 1rem;
        display: flex;
        align-items: flex-start;
        gap: 0.5rem;
    }

    .alert-success i {
        color: rgb(34, 197, 94);
        font-size: 1.25rem;
    }
</style>
