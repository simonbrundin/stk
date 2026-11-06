<template>
  <div class="start">
    <header class="start__header">
      <h1 class="start__title">Hovet</h1>
      <p class="start__subtitle">Lunchkö, bokningar och veckans meny.</p>
    </header>

    <section class="card">
      <div class="card__head">
        <div>
          <h2 class="card__title">Lunchmeny</h2>
          <p class="card__hint">
            Hämtas direkt från matochmat.se - alltså den senast publicerade menyn.
          </p>
        </div>

        <div class="week">
          <button
            class="week__step"
            type="button"
            :disabled="loading"
            aria-label="Föregående vecka"
            @click="previousWeek"
          >
            ‹
          </button>
          <span class="week__label">V{{ week || '–' }}</span>
          <button
            class="week__step"
            type="button"
            :disabled="loading"
            aria-label="Nästa vecka"
            @click="nextWeek"
          >
            ›
          </button>
        </div>
      </div>

      <button
        class="button button--primary"
        type="button"
        :disabled="loading"
        @click="refresh()"
      >
        {{ loading ? 'Hämtar menyn…' : 'Hämta ny meny' }}
      </button>

      <p v-if="error" class="alert">{{ error }}</p>

      <p v-if="fetchedAt" class="meta">
        Hämtad {{ formatGeneratedAt(fetchedAt) }} · vecka {{ week }}
      </p>

      <p v-if="!loading && days.length === 0 && !error" class="meta">
        Knappt någon meny har hämtats ännu.
      </p>

      <ul v-if="days.length" class="days">
        <li v-for="day in days" :key="day.date" class="day">
          <div class="day__head">
            <h3 class="day__title">{{ formatMenuDay(day.date, day.weekday) }}</h3>
            <span v-if="day.isToday" class="badge">Idag</span>
            <span v-if="!day.isPublished" class="badge badge--muted">
              Ej publicerad
            </span>
          </div>

          <ul v-if="day.dishes.length" class="dishes">
            <li v-for="(dish, index) in day.dishes" :key="index" class="dish">
              <span class="dish__name">{{ dish.name }}</span>
              <span v-if="dish.description" class="dish__description">
                {{ dish.description }}
              </span>
            </li>
          </ul>
        </li>
      </ul>
    </section>

    <section class="card">
      <h2 class="card__title">Publicera menyn</h2>
      <p class="card__hint">
        Sidan <NuxtLink to="/lunch">/lunch</NuxtLink> är statisk och byggs av
        GitHub Actions. Den körs automatiskt varje morgon kl. 06:00 (svensk
        tid) och hämtar då menyn på nytt. Vill du publicera direkt kan du
        starta bygget manuellt:
      </p>

      <a
        class="button button--secondary"
        href="https://github.com/simonbrundin/stk/actions/workflows/pages.yaml"
        target="_blank"
        rel="noopener noreferrer"
      >
        Starta bygget i GitHub Actions
      </a>
    </section>

    <section class="card">
      <h2 class="card__title">Övrigt</h2>
      <ul class="links">
        <li><NuxtLink to="/lunch">Lunchmeny (affisch)</NuxtLink></li>
        <li><NuxtLink to="/bookings">Bokningar</NuxtLink></li>
        <li><NuxtLink to="/health">Health check</NuxtLink></li>
      </ul>
    </section>
  </div>
</template>

<script setup lang="ts">
const { loading, error, days, week, fetchedAt, refresh, previousWeek, nextWeek } =
  useLiveLunchMenu()

onMounted(() => {
  refresh()
})
</script>

<style scoped>
.start {
  max-width: 46rem;
  margin: 0 auto;
  padding: 2rem 1.25rem 4rem;
  color: #1f2937;
  font-family: system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif;
  line-height: 1.5;
}

.start__header {
  margin-bottom: 1.5rem;
}

.start__title {
  margin: 0;
  font-size: 2rem;
  letter-spacing: -0.02em;
}

.start__subtitle {
  margin: 0.25rem 0 0;
  color: #6b7280;
}

.card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  padding: 1.25rem;
  margin-bottom: 1.25rem;
}

.card__head {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  align-items: flex-start;
  justify-content: space-between;
}

.card__title {
  margin: 0;
  font-size: 1.125rem;
}

.card__hint {
  margin: 0.25rem 0 1rem;
  color: #6b7280;
  font-size: 0.9rem;
}

.week {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.week__label {
  min-width: 3rem;
  text-align: center;
  font-weight: 600;
}

.week__step {
  width: 2rem;
  height: 2rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  background: #fff;
  font-size: 1.25rem;
  line-height: 1;
  cursor: pointer;
}

.week__step:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.button {
  display: inline-block;
  margin-top: 0.5rem;
  padding: 0.6rem 1rem;
  border: 1px solid transparent;
  border-radius: 0.5rem;
  font: inherit;
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
}

.button--primary {
  background: #111827;
  color: #fff;
}

.button--primary:hover:not(:disabled) {
  background: #1f2937;
}

.button--secondary {
  border-color: #d1d5db;
  color: #111827;
  background: #fff;
}

.button--secondary:hover {
  background: #f9fafb;
}

.button:disabled {
  opacity: 0.6;
  cursor: progress;
}

.alert {
  margin: 0.75rem 0 0;
  padding: 0.75rem;
  border-radius: 0.5rem;
  background: #fef2f2;
  color: #b91c1c;
  font-size: 0.9rem;
}

.meta {
  margin: 0.75rem 0 0;
  color: #6b7280;
  font-size: 0.85rem;
}

.days {
  list-style: none;
  margin: 1.25rem 0 0;
  padding: 0;
  display: grid;
  gap: 1rem;
}

.day__head {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.day__title {
  margin: 0;
  font-size: 1rem;
}

.badge {
  padding: 0.1rem 0.45rem;
  border-radius: 999px;
  background: #ecfdf5;
  color: #047857;
  font-size: 0.75rem;
  font-weight: 600;
}

.badge--muted {
  background: #f3f4f6;
  color: #6b7280;
}

.dishes {
  list-style: none;
  margin: 0.35rem 0 0;
  padding: 0;
  display: grid;
  gap: 0.35rem;
}

.dish {
  display: grid;
  gap: 0.1rem;
}

.dish__name {
  font-weight: 600;
}

.dish__description {
  color: #6b7280;
  font-size: 0.875rem;
}

.links {
  list-style: none;
  margin: 0.75rem 0 0;
  padding: 0;
  display: grid;
  gap: 0.35rem;
}

.links a {
  color: #2563eb;
}
</style>
