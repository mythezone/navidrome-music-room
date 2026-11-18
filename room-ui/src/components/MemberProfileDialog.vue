<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import RoomDialog from './RoomDialog.vue'
import { useRoomStore } from '../stores/room'
import { useI18n } from '../lib/i18n'
import type { MemberProfile } from '../types'
const store = useRoomStore()
const { t, errorText } = useI18n()
const profile = ref<MemberProfile>()
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const own = computed(() => store.sameUser(store.profileUsername))
const progress = computed(() => {
  if (!profile.value) return 0
  if (profile.value.level.level >= 10) return 100
  const start = profile.value.level.currentThresholdMinutes || 0
  return Math.max(0, Math.min(100, (profile.value.level.effectiveMinutes - start) / Math.max(1, profile.value.level.nextThresholdMinutes - start) * 100))
})
let request = 0
watch(() => store.profileUsername, async username => {
  const current = ++request
  profile.value = undefined
  error.value = ''
  if (!username) return
  loading.value = true
  try { const result = await store.memberProfile(username); if (current === request) profile.value = result }
  catch (cause) { if (current === request) error.value = errorText(cause) }
  finally { if (current === request) loading.value = false }
})
async function select(id: string): Promise<void> {
  if (!profile.value || saving.value || !own.value) return
  saving.value = true
  try { await store.selectBadge(id); profile.value.selectedBadgeID = id }
  catch (cause) { error.value = errorText(cause) }
  finally { saving.value = false }
}
function leave(): void { if (window.confirm(t('confirmLeave'))) void store.leaveRoom() }
</script>
<template>
  <RoomDialog :open="!!store.profileUsername" :title="t('profile')" @close="store.profileUsername = ''">
    <div v-if="loading" class="panel-loading">{{ t('loading') }}</div>
    <p v-if="error" class="media-error" role="alert">{{ error }}</p>
    <template v-if="profile">
      <div class="profile-identity"><span class="profile-avatar">{{ (profile.displayName || profile.username).slice(0, 1).toUpperCase() }}</span><div><h3>{{ profile.displayName || profile.username }}</h3><span class="count-pill">{{ t(profile.role) }}</span><p>{{ t('level', { level: profile.level.level }) }}</p></div></div>
      <progress class="level-progress" :value="progress" max="100" :aria-label="t('level', { level: profile.level.level })" /><p>{{ t(profile.level.level >= 10 ? 'maxLevel' : 'nextLevel', { count: Math.max(0, Math.ceil(profile.level.nextThresholdMinutes - profile.level.effectiveMinutes)) }) }}</p>
      <div class="stat-grid"><div><strong>{{ profile.stats.listenedCount }}</strong><span>{{ t('listened') }}</span></div><div><strong>{{ profile.stats.requestedCount }}</strong><span>{{ t('requested') }}</span></div><div><strong>{{ profile.stats.chatCount }}</strong><span>{{ t('chatCount') }}</span></div><div><strong>{{ t('minutes', { count: Math.floor(profile.stats.staySeconds / 60) }) }}</strong><span>{{ t('stayTime') }}</span></div></div>
      <h3>{{ t('badges') }}</h3><div class="badge-grid"><button v-for="badge in profile.badges || []" :key="badge.badgeID" :class="['badge-card', { selected: profile.selectedBadgeID === badge.badgeID }]" :disabled="!own || !badge.unlocked || saving" :aria-pressed="profile.selectedBadgeID === badge.badgeID" @click="select(profile.selectedBadgeID === badge.badgeID ? '' : badge.badgeID)"><span class="badge-emoji">{{ '🏅' }}</span><strong>{{ t(`badge.${badge.kind}`, { count: badge.target }) }}<span v-if="badge.subject"> · {{ badge.subject }}</span></strong><small>{{ t(badge.unlocked ? 'unlocked' : 'locked') }} · {{ badge.count }} / {{ badge.target }}</small><progress :value="badge.count" :max="Math.max(1, badge.target)" :aria-label="t('badges')" /></button></div>
      <button v-if="own && profile.selectedBadgeID" class="text-link" :disabled="saving" @click="select('')">{{ t('clearBadge') }}</button>
      <div class="dialog-actions"><button class="secondary-button" @click="store.suggestUserToChat(profile)">{{ t('mention') }}</button><button v-if="own && profile.role !== 'owner'" class="secondary-button danger" @click="leave">{{ t('leave') }}</button></div>
    </template>
  </RoomDialog>
</template>
