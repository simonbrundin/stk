<template>
  <div class="lunch">
    <header class="lunch__header">
      <h1 class="lunch__title">Lunch</h1>
      <p class="lunch__restaurant">
        <a :href="menu.restaurant.url" target="_blank" rel="noopener">
          {{ menu.restaurant.name }}
        </a>
      </p>
      <p v-if="menu.generatedAt" class="lunch__updated">
        Menyn är hämtad {{ formatGeneratedAt(menu.generatedAt) }}
      </p>
    </header>

    <p v-if="menu.days.length === 0" class="lunch__empty">
      Menyn kunde inte hämtas. Försök igen senare.
    </p>

    <section v-for="day in menu.days" :key="day.date" class="day">
      <h2 class="day__title">
        {{ formatMenuDate(day.date) }}
        <span v-if="day.isToday" class="day__badge">idag</span>
      </h2>

      <p v-if="!day.isPublished" class="day__unpublished">
        Menyn för dagen är inte publicerad ännu.
      </p>

      <ul v-else class="dishes">
        <li v-for="(dish, index) in day.dishes" :key="index" class="dish">
          <div class="dish__topRow">
            <span class="dish__name">{{ dish.name }}</span>
            <span v-if="dish.price" class="dish__price">{{ dish.price }} kr</span>
          </div>
          <p v-if="dish.description" class="dish__description">
            {{ dish.description }}
          </p>
          <ul v-if="dish.tags.length > 0" class="dish__tags">
            <li v-for="tag in dish.tags" :key="tag" class="dish__tag">
              {{ tag }}
            </li>
          </ul>
        </li>
      </ul>
    </section>

    <footer class="lunch__footer">
      <p>
        Menyn kommer från
        <a :href="menu.restaurant.url" target="_blank" rel="noopener">Mat och Mat</a>.
      </p>
    </footer>
  </div>
</template>

<script setup lang="ts">
const menu = useLunchMenu()
</script>

<style scoped>
.lunch {
  max-width: 44rem;
  margin: 0 auto;
  padding: 1.5rem 1.25rem 3rem;
}

.lunch__title {
  margin: 0 0 0.25rem;
  font-size: 1.75rem;
}

.lunch__restaurant {
  margin: 0 0 0.25rem;
  font-size: 1rem;
}

.lunch__updated,
.lunch__empty {
  margin: 0;
  color: var(--color-muted, #64748b);
  font-size: 0.875rem;
}

.day {
  margin-top: 2rem;
}

.day__title {
  margin: 0 0 0.75rem;
  font-size: 1.25rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.day__badge {
  padding: 0.05rem 0.5rem;
  border-radius: 999px;
  background: var(--color-accent, #2563eb);
  color: #fff;
  font-size: 0.75rem;
  font-weight: 600;
}

.day__unpublished {
  margin: 0;
  color: var(--color-muted, #64748b);
  font-style: italic;
}

.dishes {
  margin: 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 0.75rem;
}

.dish {
  padding: 0.75rem 1rem;
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 0.5rem;
}

.dish__topRow {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
}

.dish__name {
  font-weight: 600;
}

.dish__price {
  font-weight: 600;
  white-space: nowrap;
}

.dish__description {
  margin: 0.25rem 0 0;
  color: var(--color-muted, #64748b);
  font-size: 0.9rem;
}

.dish__tags {
  margin: 0.5rem 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.dish__tag {
  padding: 0.05rem 0.5rem;
  border-radius: 999px;
  background: #ecfdf5;
  color: #15803d;
  font-size: 0.75rem;
  font-weight: 600;
}

.lunch__footer {
  margin-top: 3rem;
  padding-top: 1rem;
  border-top: 1px solid var(--color-border, #e2e8f0);
  color: var(--color-muted, #64748b);
  font-size: 0.875rem;
}

.lunch__footer p {
  margin: 0;
}
</style>
