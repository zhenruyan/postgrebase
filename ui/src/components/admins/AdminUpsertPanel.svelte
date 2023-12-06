<script>
    import tooltip from "@/actions/tooltip";
    import Field from "@/components/base/Field.svelte";
    import OverlayPanel from "@/components/base/OverlayPanel.svelte";
    import Toggler from "@/components/base/Toggler.svelte";
    import { confirm } from "@/stores/confirmation";
    import { setErrors } from "@/stores/errors";
    import { addSuccessToast } from "@/stores/toasts";
    import ApiClient from "@/utils/ApiClient";
    import CommonHelper from "@/utils/CommonHelper";
    import { Admin } from "pocketbase";
    import { createEventDispatcher } from "svelte";
    import { slide } from "svelte/transition";

    const dispatch = createEventDispatcher();
    const formId = "admin_" + CommonHelper.randomString(5);

    let panel;
    let admin = new Admin();
    let isSaving = false;
    let confirmClose = false; // prevent close recursion
    let avatar = 0;
    let email = "";
    let password = "";
    let passwordConfirm = "";
    let changePasswordToggle = false;

    $: hasChanges =
        (admin.$isNew && email != "") ||
        changePasswordToggle ||
        email !== admin.email ||
        avatar !== admin.avatar;

    export function show(model) {
        load(model);

        confirmClose = true;

        return panel?.show();
    }

    export function hide() {
        return panel?.hide();
    }

    function load(model) {
        admin = model?.$clone ? model.$clone() : new Admin();
        reset(); // reset form
    }

    function reset() {
        changePasswordToggle = false;
        email = admin?.email || "";
        avatar = admin?.avatar || 0;
        password = "";
        passwordConfirm = "";
        setErrors({}); // reset errors
    }

    function save() {
        if (isSaving || !hasChanges) {
            return;
        }

        isSaving = true;

        const data = { email, avatar };
        if (admin.$isNew || changePasswordToggle) {
            data["password"] = password;
            data["passwordConfirm"] = passwordConfirm;
        }

        let request;
        if (admin.$isNew) {
            request = ApiClient.admins.create(data);
        } else {
            request = ApiClient.admins.update(admin.id, data);
        }

        request
            .then(async (result) => {
                confirmClose = false;
                hide();
                addSuccessToast(admin.$isNew ? "成功创建管理员" : "成功更新管理员");

                if (ApiClient.authStore.model?.id === result.id) {
                    ApiClient.authStore.save(ApiClient.authStore.token, result);
                }

                dispatch("save", result);
            })
            .catch((err) => {
                ApiClient.error(err);
            })
            .finally(() => {
                isSaving = false;
            });
    }

    function deleteConfirm() {
        if (!admin?.id) {
            return; // nothing to delete
        }

        confirm(`是否删除选中的管理员`, () => {
            return ApiClient.admins
                .delete(admin.id)
                .then(() => {
                    confirmClose = false;
                    hide();
                    addSuccessToast("成功删除管理员");
                    dispatch("delete", admin);
                })
                .catch((err) => {
                    ApiClient.error(err);
                });
        });
    }
</script>

<OverlayPanel
    bind:this={panel}
    popup
    class="admin-panel"
    beforeHide={() => {
        if (hasChanges && confirmClose) {
            confirm("是否不保存而直接关闭面板", () => {
                confirmClose = false;
                hide();
            });
            return false;
        }
        return true;
    }}
    on:hide
    on:show
>
    <svelte:fragment slot="header">
        <h4>
            {admin.$isNew ? "新建管理员" : "修改管理员"}
        </h4>
    </svelte:fragment>

    <form id={formId} class="grid" autocomplete="off" on:submit|preventDefault={save}>
        {#if !admin.$isNew}
            <Field class="form-field readonly" name="id" let:uniqueId>
                <label for={uniqueId}>
                    <i class={CommonHelper.getFieldTypeIcon("primary")} />
                    <span class="txt">id</span>
                </label>
                <div class="form-field-addon">
                    <i
                        class="ri-calendar-event-line txt-disabled"
                        use:tooltip={{
                            text: `Created: ${admin.created}\nUpdated: ${admin.updated}`,
                            position: "left",
                        }}
                    />
                </div>
                <input type="text" id={uniqueId} value={admin.id} readonly />
            </Field>
        {/if}

        <div class="content">
            <p class="section-title">头像</p>
            <div class="flex flex-gap-xs flex-wrap">
                {#each [0, 1, 2, 3, 4, 5, 6, 7, 8, 9] as index}
                    <button
                        type="button"
                        class="link-fade thumb thumb-circle {index == avatar ? 'thumb-active' : 'thumb-sm'}"
                        on:click={() => (avatar = index)}
                    >
                        <img
                            src="{import.meta.env.BASE_URL}images/avatars/avatar{index}.svg"
                            alt="Avatar {index}"
                        />
                    </button>
                {/each}
            </div>
        </div>

        <Field class="form-field required" name="email" let:uniqueId>
            <label for={uniqueId}>
                <i class={CommonHelper.getFieldTypeIcon("email")} />
                <span class="txt">邮箱</span>
            </label>
            <input type="email" autocomplete="off" id={uniqueId} required bind:value={email} />
        </Field>

        {#if !admin.$isNew}
            <Field class="form-field form-field-toggle" let:uniqueId>
                <input type="checkbox" id={uniqueId} bind:checked={changePasswordToggle} />
                <label for={uniqueId}>修改密码</label>
            </Field>
        {/if}

        {#if admin.$isNew || changePasswordToggle}
            <div class="col-12">
                <div class="grid" transition:slide|local={{ duration: 150 }}>
                    <div class="col-sm-6">
                        <Field class="form-field required" name="password" let:uniqueId>
                            <label for={uniqueId}>
                                <i class="ri-lock-line" />
                                <span class="txt">密码</span>
                            </label>
                            <input
                                type="password"
                                autocomplete="new-password"
                                id={uniqueId}
                                required
                                bind:value={password}
                            />
                        </Field>
                    </div>
                    <div class="col-sm-6">
                        <Field class="form-field required" name="passwordConfirm" let:uniqueId>
                            <label for={uniqueId}>
                                <i class="ri-lock-line" />
                                <span class="txt">密码验证</span>
                            </label>
                            <input
                                type="password"
                                autocomplete="new-password"
                                id={uniqueId}
                                required
                                bind:value={passwordConfirm}
                            />
                        </Field>
                    </div>
                </div>
            </div>
        {/if}
    </form>

    <svelte:fragment slot="footer">
        {#if !admin.$isNew}
            <button type="button" aria-label="More" class="btn btn-sm btn-circle btn-transparent">
                <!-- empty span for alignment -->
                <span />
                <i class="ri-more-line" />
                <Toggler class="dropdown dropdown-upside dropdown-left dropdown-nowrap">
                    <button type="button" class="dropdown-item txt-danger" on:click={() => deleteConfirm()}>
                        <i class="ri-delete-bin-7-line" />
                        <span class="txt">删除</span>
                    </button>
                </Toggler>
            </button>
            <div class="flex-fill" />
        {/if}

        <button type="button" class="btn btn-transparent" disabled={isSaving} on:click={() => hide()}>
            <span class="txt">取消</span>
        </button>
        <button
            type="submit"
            form={formId}
            class="btn btn-expanded"
            class:btn-loading={isSaving}
            disabled={!hasChanges || isSaving}
        >
            <span class="txt">{admin.$isNew ? "创建" : "保存更改"}</span>
        </button>
    </svelte:fragment>
</OverlayPanel>
