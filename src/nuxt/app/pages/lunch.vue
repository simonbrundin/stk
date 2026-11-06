<template>
  <div class="lunch" :style="{ zoom: scale }">
    <section v-for="(screen, index) in screens" :key="index" class="screen">
      <header class="screen__header">
        <h1 class="screen__title">Lunchmeny</h1>
        <p class="screen__week">V{{ week }}</p>
      </header>

      <div class="screen__days">
        <section v-for="day in screen" :key="day.date" class="day">
          <h2 class="day__title">{{ formatMenuDay(day.date, day.weekday) }}</h2>

          <p v-if="!day.isPublished" class="day__unpublished">
            Menyn för dagen är inte publicerad ännu.
          </p>

          <ul v-else class="day__dishes">
            <li v-for="(dish, dishIndex) in day.dishes" :key="dishIndex" class="dish">
              <p class="dish__name">
                <span>{{ dish.name }}</span>
                <span v-if="dish.price" class="dish__leader" aria-hidden="true" />
              </p>
              <p v-if="dish.description" class="dish__description">{{ dish.description }}</p>
              <p v-if="dish.tags.length" class="dish__tags">{{ dish.tags.join(', ') }}</p>
            </li>
          </ul>
        </section>
      </div>

      <img
        class="screen__logo"
        :src="logoSrc"
        :alt="menu.restaurant.name"
        width="280"
        height="131"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
const menu = useLunchMenu()
const runtimeConfig = useRuntimeConfig()

/** ISO-veckan som menyn tillhör, t.ex. 45. */
const week = computed(() => isoWeekNumber(menu.days[0]?.date ?? ''))

/** Logotypen ligger i public/ och måste följa baseURL (t.ex. /stk/). */
const logoSrc = computed(() => `${runtimeConfig.app.baseURL}images/hovet-logo.png`)

/** Menyn delas upp på skärmar som i mockupen. */
const screens = computed(() => groupDaysIntoScreens(menu.days))

/**
 * Skärmen är 1080x1920 px. På smalare skärmar skalas hela skärmen ner,
 * så att layouten alltid är den som mätts mot mockupen.
 */
const scale = ref(1)

function updateScale(): void {
  scale.value = Math.min(1, window.innerWidth / 1080)
}

onMounted(() => {
  updateScale()
  window.addEventListener('resize', updateScale)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateScale)
})
</script>

<style scoped>
/* Skärmen är en fast 1080x1920-kanvas, så webbläsarens standardmarginal
   på body måste nollställas. */
:global(html),
:global(body) {
  margin: 0;
  padding: 0;
  background: #e6e6e6;
}

.lunch {
  --lunch-black: #000;
  --lunch-grey: #767477;
  --lunch-muted: #a3a3a3;
  --lunch-tag: #8c8c8c;
  /* Textens vänsterkant ligger 2 px innanför kolumnens kant (sidbäring). */
  --lunch-text-inset: 2px;

  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
  min-height: 100vh;
  background: #e6e6e6;
  color: var(--lunch-black);
  font-family: 'Inter', system-ui, -apple-system, 'Segoe UI', Roboto, Arial, sans-serif;
  font-feature-settings: 'kern';
  -webkit-font-smoothing: antialiased;
}

/* --------------------------------------------------------------- Skärm */

.screen {
  position: relative;
  width: 1080px;
  min-height: 1920px;
  background: #fff;
  overflow: hidden;
}

.screen__header {
  display: flex;
  justify-content: center;
  align-items: baseline;
  gap: 21px;
  padding-top: 80px;
}

.screen__title {
  margin: 0;
  color: var(--lunch-grey);
  font-family: 'Archivo', 'Inter', sans-serif;
  font-size: 85px;
  font-variation-settings: 'wdth' 77, 'wght' 700;
  letter-spacing: -0.02em;
  line-height: 1;
}

.screen__week {
  margin: 0;
  color: var(--lunch-grey);
  font-family: 'Archivo', 'Inter', sans-serif;
  font-size: 85px;
  font-variation-settings: 'wdth' 73, 'wght' 800;
  letter-spacing: -0.01em;
  line-height: 1;
}

/* ------------------------------------------------------------- Innehåll */

.screen__days {
  position: absolute;
  top: 244px;
  left: 181px;
  width: 740px;
}

.day {
  margin-bottom: 19px;
}

.day__title {
  margin: 0 0 11px;
  font-family: 'Archivo', 'Inter', sans-serif;
  font-size: 28px;
  font-variation-settings: 'wdth' 78, 'wght' 800;
  letter-spacing: -0.01em;
  line-height: 1.1;
}

/* Tunn linje under dagrubriken, som i mockupen. Linjen börjar vid
   textens vänsterkant (sidbäringen på 2 px) och slutar vid kolumnens kant. */
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
  font-size: 18px;
  font-style: italic;
}

.day__dishes {
  margin: 0;
  padding: 0;
  list-style: none;
}

.dish {
  margin-bottom: 17px;
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
  font-size: 19px;
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
  margin: 2px 0 0;
  color: var(--lunch-grey);
  font-size: 18px;
  font-weight: 400;
  line-height: 1.35;
}

/* Taggar (t.ex. "Vegetarisk") visas som en liten indrad rad under rätten. */
.dish__tags {
  margin: 5px 0 0 18px;
  color: var(--lunch-tag);
  font-size: 13px;
  letter-spacing: 0.01em;
  line-height: 1.2;
}

/* --------------------------------------------------------------- Logotyp */

/* Logotypen är en beskärning av mockupens sidfot (x 574-972, y 1681-1868)
   och placeras därför i samma nedre högra hörn, men i mindre skala. */
.screen__logo {
  position: absolute;
  right: 108px;
  bottom: 52px;
  width: 280px;
  height: 131px;
  mix-blend-mode: multiply;
}
</style>
