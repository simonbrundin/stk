<template>
  <div>
    <NuxtRouteAnnouncer />
    <nav class="nav">
      <NuxtLink to="/">Start</NuxtLink>
      <NuxtLink to="/bookings">Bokningar</NuxtLink>
      <NuxtLink to="/lunch">Lunch</NuxtLink>
    </nav>
    <NuxtPage />
  </div>
</template>

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
