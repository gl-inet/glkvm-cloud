<!--
 * @Description: Wraps gl-web-main's BaseModal with project-i18n defaults for
 *   okText / cancelText. The upstream BaseModal falls back to its own internal
 *   i18n table when these props are omitted, but that table only contains
 *   zh/en — any other locale throws at render time and the entire footer slot
 *   fails to mount, so Confirm/Cancel buttons disappear in de/es/fr/ja/ko.
 *   Defaulting the props here routes labels through the project's vue-i18n,
 *   which knows every supported locale.
-->
<template>
    <BaseModal
        :cancelText="cancelText"
        :okText="okText"
    >
        <template v-for="(_, name) in $slots" #[name]="slotData">
            <slot :name="name" v-bind="slotData || {}" />
        </template>
    </BaseModal>
</template>

<script setup lang="ts">
import { BaseModal } from 'gl-web-main/components'

withDefaults(defineProps<{
    okText?: string
    cancelText?: string
}>(), {
    okText: 'common.confirm',
    cancelText: 'common.cancel',
})
</script>
