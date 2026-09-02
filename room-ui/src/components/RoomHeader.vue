<script setup lang="ts">
import { ref } from 'vue'
import { useRoomStore } from '../stores/room'

const emit = defineEmits<{ share: [] }>()
const store = useRoomStore()
const memberOpen = ref(false)
</script>

<template>
  <header class="room-header">
    <div class="room-identity">
      <div class="room-avatar"><i class="mdi mdi-radio" /></div>
      <div>
        <h1>{{ store.room?.name || '一起听歌' }}</h1>
        <p>
          <span :class="['connection-dot', { offline: !store.connected }]" />
          {{ store.connected ? '跟随房间播放' : store.reconnecting ? '正在重新连接' : '连接中' }}
        </p>
      </div>
    </div>
    <div class="header-actions">
      <button class="header-action presence-button" type="button" @click="memberOpen = !memberOpen">
        <i class="mdi mdi-account-multiple-outline" />
        <span>{{ store.onlineCount }} 人在线</span>
      </button>
      <button class="header-action" type="button" @click="emit('share')"><i class="mdi mdi-share-variant-outline" /><span>分享</span></button>
      <a v-if="store.deepLink" class="header-action app-link" :href="store.deepLink"><i class="mdi mdi-cellphone-music" /><span>MusicMate</span></a>
      <div class="user-avatar" :title="store.session?.user.displayName || store.session?.user.username">{{ (store.session?.user.displayName || store.session?.user.username || '?').slice(0, 1).toUpperCase() }}</div>
    </div>
    <aside v-if="memberOpen" class="presence-popover">
      <div class="popover-title"><strong>房间成员</strong><button class="icon-button" aria-label="关闭成员列表" @click="memberOpen = false"><i class="mdi mdi-close" /></button></div>
      <ul>
        <li v-for="member in store.members" :key="member.username">
          <span :class="['member-dot', { offline: !member.online }]" />
          <span><strong>{{ member.displayName || member.username }}</strong><small>@{{ member.username }} · {{ member.role === 'owner' ? '房主' : '成员' }}</small></span>
        </li>
      </ul>
    </aside>
  </header>
</template>
