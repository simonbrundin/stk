<template>
  <div>
    <NuxtRouteAnnouncer />
    <nav v-if="!isStandalone" class="nav">
      <NuxtLink to="/">Start</NuxtLink>
      <NuxtLink to="/bookings">Bokningar</NuxtLink>
      <NuxtLink to="/lunch">Lunch</NuxtLink>
    </nav>
    <NuxtPage />
  </div>
</template>

<script setup lang="ts">
// Lunchsidan är en fristående meny (designad som ett affisch) och ska inte
// visa appens navigation.
const route = useRoute()
const isStandalone = computed(() => route.path.replace(/\/$/, '') === '/lunch')
</script>

<style scoped>
.nav {
  display: flex;
  gap: 1rem;
  padding: 0.75rem 1.25rem;
  border-bottom: 1px solid var(--color-border, #e2e8f0);
}

.nav a {
  color: var(--color-muted, #64748b);
  text-decoration: none;
  font-weight: 500;
}

.nav a.router-link-active {
  color: var(--color-accent, #2563eb);
}
</style>
