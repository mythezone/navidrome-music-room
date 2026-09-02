<script setup lang="ts">
defineProps<{ feature?: string }>()
</script>

<template>
  <section class="work-panel coming-soon-panel" aria-labelledby="coming-soon-title">
    <div class="coming-orbit"><i class="mdi mdi-forum-outline" /></div>
    <span class="section-kicker">OPEN SOURCE ROADMAP</span>
    <h2 id="coming-soon-title">{{ feature || '这个功能' }}正在开发</h2>
    <p>同步听歌、点歌、邀请和成员权限已经可用。这个功能会继续以开源方式完善，欢迎在 GitHub 提交建议或参与贡献。</p>
    <span class="roadmap-chip"><i class="mdi mdi-source-branch" /> 欢迎贡献</span>
  </section>
</template>
