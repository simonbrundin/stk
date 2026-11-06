<template>
  <div class="lunch" :style="{ zoom: scale }">
    <section class="screen">
      <header class="screen__header">
        <h1 class="screen__title">Lunchmeny</h1>
        <p class="screen__week">V{{ week }}</p>
      </header>

      <div class="screen__columns">
        <section class="column">
          <section v-for="day in layout.left" :key="day.date" class="day">
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
        </section>

        <section class="column">
          <section v-for="day in layout.right" :key="day.date" class="day">
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

          <section
            v-if="layout.soup || layout.vegetarian"
            class="weekly"
          >
            <div v-if="layout.soup" class="weekly__item">
              <h2 class="weekly__title">{{ layout.soup.name }}</h2>
              <p v-if="layout.soup.description" class="weekly__description">
                {{ layout.soup.description }}
              </p>
            </div>

            <div v-if="layout.vegetarian" class="weekly__item">
              <h2 class="weekly__title">{{ layout.vegetarian.name }}</h2>
              <p v-if="layout.vegetarian.description" class="weekly__description">
                {{ layout.vegetarian.description }}
              </p>
            </div>
          </section>
        </section>
      </div>

      <img
        class="screen__logo"
        :src="logoSrc"
        :alt="menu.restaurant.name"
        width="240"
        height="113"
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

/** Veckans dagar i två kolumner, med soppa och vegetarisk rätt längst ner. */
const layout = computed(() => buildWeekLayout(menu.days))

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
  height: 1920px;
  overflow: hidden;
  background: #fff;
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

/* ------------------------------------------------------------- Kolumner */

/* Två kolumner: måndag–torsdag till vänster, fredag–söndag till höger. */
.screen__columns {
  position: absolute;
  top: 244px;
  left: 100px;
  right: 100px;
  display: flex;
  align-items: flex-start;
  gap: 60px;
}

.column {
  flex: 1 1 0;
  min-width: 0;
}

/* ------------------------------------------------------------- Innehåll */

.day {
  margin-bottom: 26px;
}

.day__title {
  margin: 0 0 11px;
  font-family: 'Archivo', 'Inter', sans-serif;
  font-size: 26px;
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
  margin-bottom: 16px;
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
  font-size: 17px;
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

/* ----------------------------------------------------- Veckans highlight */

/* Soppan och den vegetariska rätten gäller hela veckan och ligger därför
   under söndag i högra kolumnen. */
.weekly {
  margin-top: 44px;
  padding-top: 20px;
  border-top: 2px solid var(--lunch-grey);
}

.weekly__item + .weekly__item {
  margin-top: 16px;
}

.weekly__title {
  margin: 0;
  font-family: 'Archivo', 'Inter', sans-serif;
  font-size: 26px;
  font-variation-settings: 'wdth' 78, 'wght' 800;
  letter-spacing: -0.01em;
  line-height: 1.1;
}

.weekly__description {
  margin: 4px 0 0;
  color: var(--lunch-grey);
  font-size: 17px;
  line-height: 1.35;
}

/* --------------------------------------------------------------- Logotyp */

/* Logotypen är en beskärning av mockupens sidfot (x 574-972, y 1681-1868)
   och placeras därför i samma nedre högra hörn, i 60 % skala (240x113 px). */
.screen__logo {
  position: absolute;
  right: 108px;
  bottom: 52px;
  width: 240px;
  height: 113px;
  mix-blend-mode: multiply;
}
</style>
