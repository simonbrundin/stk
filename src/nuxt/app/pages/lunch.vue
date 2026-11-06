<template>
  <div class="lunch">
    <header class="lunch__header">
      <h1 class="lunch__title">Lunchmeny</h1>
      <p class="lunch__week">V{{ week }}</p>
    </header>

    <main class="lunch__body">
      <section v-for="day in menu.days" :key="day.date" class="day">
        <h2 class="day__title">{{ formatMenuDay(day.date, day.weekday) }}</h2>

        <p v-if="!day.isPublished" class="day__unpublished">
          Menyn för dagen är inte publicerad ännu.
        </p>

        <ul v-else class="day__dishes">
          <li v-for="(dish, index) in day.dishes" :key="index" class="dish">
            <p class="dish__name">
              <span>{{ dish.name }}</span>
              <span v-if="dish.price" class="dish__leader" aria-hidden="true" />
            </p>
            <p v-if="dish.description" class="dish__description">{{ dish.description }}</p>
          </li>
        </ul>
      </section>
    </main>

    <footer class="lunch__footer">
      <div class="footer__mark" aria-hidden="true">
        <span>H</span>
      </div>

      <div class="footer__wordmark">
        <span class="wordmark__rule wordmark__rule--top" aria-hidden="true" />
        <p class="wordmark__name">{{ wordmark.name }}</p>
        <p class="wordmark__sub">{{ wordmark.sub }}</p>
        <span class="wordmark__rule wordmark__rule--bottom" aria-hidden="true" />
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
const menu = useLunchMenu()

/** ISO-veckan som menyn tillhör, t.ex. 45. */
const week = computed(() => isoWeekNumber(menu.days[0]?.date ?? ''))

/** Namnet splittat i en stor logotyptext och en undertext. */
const wordmark = computed(() => {
  const parts = menu.restaurant.name.split(' ')

  return {
    name: parts[0]?.toUpperCase() ?? '',
    sub: parts.slice(1).join(' ').toUpperCase(),
  }
})
</script>

<style scoped>
.lunch {
  --lunch-black: #000;
  --lunch-grey: #767477;
  --lunch-muted: #a3a3a3;
  /* Textens vänsterkant ligger 2 px innanför kolumnens kant (sidbäring). */
  --lunch-text-inset: 2px;

  min-height: 100vh;
  padding: 0 clamp(1.25rem, 5vw, 2rem);
  background: #fff;
  color: var(--lunch-black);
  font-family: 'Inter', system-ui, -apple-system, 'Segoe UI', Roboto, Arial, sans-serif;
  font-feature-settings: 'kern';
  -webkit-font-smoothing: antialiased;
}

@media (min-width: 900px) {
  .lunch {
    padding: 0;
  }
}

/* --------------------------------------------------------------- Toppbar */

.lunch__header {
  display: flex;
  justify-content: center;
  align-items: baseline;
  gap: clamp(0.75rem, 2vw, 1.75rem);
  padding: clamp(1.75rem, 6.6vw, 5rem) 0 clamp(1.75rem, 7.5vw, 5.3rem);
}

.lunch__title {
  margin: 0;
  color: var(--lunch-grey);
  font-family: 'Archivo', 'Inter', sans-serif;
  font-size: clamp(2.25rem, 7.9vw, 5.3rem);
  font-variation-settings: 'wdth' 77, 'wght' 700;
  letter-spacing: -0.02em;
  line-height: 1;
}

.lunch__week {
  margin: 0;
  color: var(--lunch-grey);
  font-family: 'Archivo', 'Inter', sans-serif;
  font-size: clamp(2rem, 7.9vw, 5.3rem);
  font-variation-settings: 'wdth' 73, 'wght' 800;
  letter-spacing: -0.01em;
  line-height: 1;
}

/* -------------------------------------------------------------- Innehåll */

.lunch__body {
  max-width: 740px;
}

@media (min-width: 900px) {
  .lunch__body {
    margin-left: 16.3%;
  }
}

.day {
  margin-bottom: clamp(1rem, 1.85vw, 1.5rem);
}

.day__title {
  margin: 0 0 clamp(0.55rem, 1vw, 0.8rem);
  font-family: 'Archivo', 'Inter', sans-serif;
  font-size: clamp(1.05rem, 2.6vw, 1.75rem);
  font-variation-settings: 'wdth' 78, 'wght' 800;
  letter-spacing: -0.01em;
  line-height: 1.1;
}

/* Tunn linje under dagrubriken, som i mockupen. Linjen börjar vid
   textens vänsterkant (sidbäringen på 7 px) och slutar vid kolumnens kant. */
.day__title::after {
  content: '';
  display: block;
  height: 2px;
  margin-top: 8px;
  margin-left: var(--lunch-text-inset);
  width: calc(100% - var(--lunch-text-inset));
  background: var(--lunch-grey);
}

.day__unpublished {
  margin: 0;
  color: var(--lunch-muted);
  font-size: clamp(0.85rem, 1.5vw, 1rem);
  font-style: italic;
}

.day__dishes {
  margin: 0;
  padding: 0;
  list-style: none;
}

.dish {
  margin-bottom: clamp(0.8rem, 1.6vw, 1.3rem);
}

/* Sista rätten i en dag ska inte trycka dagens avslut neråt. */
.dish:last-child {
  margin-bottom: 0;
}

.dish__name {
  display: flex;
  align-items: baseline;
  gap: 0.4em;
  margin: 0;
  font-family: 'Archivo', 'Inter', sans-serif;
  font-size: clamp(0.9rem, 1.72vw, 1.2rem);
  font-variation-settings: 'wdth' 100, 'wght' 700;
  line-height: 1.2;
}

/* Prickledningen visar var rättens pris ska stå. I mockupen visas bara
   prickarna - själva priset saknas i bilden. */
.dish__leader {
  flex: 1;
  border-bottom: 2px dotted var(--lunch-muted);
  transform: translateY(0.15em);
}

.dish__description {
  margin: 0.15rem 0 0;
  color: var(--lunch-grey);
  font-size: clamp(0.8rem, 1.67vw, 1.15rem);
  font-weight: 400;
  line-height: 1.35;
}

/* ---------------------------------------------------------------- Footer */

.lunch__footer {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 2rem;
  margin-top: clamp(3rem, 6vw, 5rem);
  padding-bottom: clamp(2rem, 5vw, 4rem);
}

@media (min-width: 900px) {
  .lunch__footer {
    margin-left: 15%;
    margin-right: 10%;
  }
}

.footer__mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: clamp(5rem, 14.3vw, 9.6rem);
  height: clamp(5rem, 14.3vw, 9.6rem);
  border: clamp(3px, 0.6vw, 6px) solid var(--lunch-black);
  border-radius: 50%;
  color: var(--lunch-black);
  font-family: 'Archivo', 'Inter', sans-serif;
  font-size: clamp(2.5rem, 6.5vw, 4.5rem);
  font-variation-settings: 'wdth' 80, 'wght' 800;
  line-height: 1;
}

.footer__wordmark {
  text-align: left;
}

.wordmark__rule {
  display: block;
  width: 180px;
  height: 2px;
  background: var(--lunch-black);
}

.wordmark__rule--top {
  margin: 0 0 10px auto;
}

.wordmark__rule--bottom {
  margin: 16px 0 0;
}

.wordmark__name {
  margin: 0;
  font-family: 'Archivo', 'Inter', sans-serif;
  font-size: clamp(2.5rem, 12.5vw, 8.5rem);
  font-variation-settings: 'wdth' 72, 'wght' 800;
  letter-spacing: 0.06em;
  line-height: 1;
}

.wordmark__sub {
  margin: 0.5rem 0 0;
  font-size: clamp(0.6rem, 2.3vw, 1.5rem);
  font-weight: 600;
  letter-spacing: 0.28em;
  line-height: 1.2;
}
</style>
