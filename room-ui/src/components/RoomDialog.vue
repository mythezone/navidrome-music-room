<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from '../lib/i18n'
const props = defineProps<{ open: boolean; title: string; wide?: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const dialog = ref<HTMLDialogElement>()
let previousFocus: HTMLElement | null = null
watch(() => props.open, open => {
  if (!dialog.value) return
  if (open && !dialog.value.open) {
    previousFocus = document.activeElement as HTMLElement
    dialog.value.showModal()
  } else if (!open && dialog.value.open) {
    dialog.value.close()
    previousFocus?.focus()
  }
}, { immediate: true, flush: 'post' })
onBeforeUnmount(() => { dialog.value?.close(); previousFocus?.focus() })
</script>
<template>
  <dialog ref="dialog" :class="['dialog', { 'dialog-wide': wide }]" :aria-label="title" @cancel.prevent="emit('close')" @click="($event.target === dialog) && emit('close')">
    <div class="dialog-body" v-if="open">
      <header><h2>{{ title }}</h2><button class="icon-button" :aria-label="t('close')" @click="emit('close')"><i aria-hidden="true" class="mdi mdi-close" /></button></header>
      <slot />
    </div>
  </dialog>
</template>
