<script setup lang="ts">
import { useI18n, type Locale } from '../lib/i18n'
const { t, locale, locales, localeNames, setLocale } = useI18n()
</script>
<template>
  <select class="language-selector" :value="locale" :aria-label="t('language')" @change="setLocale(($event.target as HTMLSelectElement).value as Locale)">
    <option v-for="code in locales" :key="code" :value="code">{{ localeNames[code] }}</option>
  </select>
</template>
