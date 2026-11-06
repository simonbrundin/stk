<template>
  <div class="container">
    <h1>Health Check</h1>
    
    <div v-if="loading" class="loading">Loading...</div>
    
    <div v-else-if="error" class="error">
      <p>Error: {{ error }}</p>
      <button @click="fetchHealth">Retry</button>
    </div>
    
    <div v-else-if="healthData" class="health-data">
      <p><strong>Status:</strong> {{ healthData.status }}</p>
      <p><strong>Version:</strong> {{ healthData.version || 'N/A' }}</p>
      <p><strong>Timestamp:</strong> {{ healthData.timestamp }}</p>
      <button @click="fetchHealth">Refresh</button>
    </div>
  </div>
</template>

<script setup lang="ts">
const loading = ref(true)
const error = ref<string | null>(null)
const healthData = ref<{
  status: string
  version?: string
  timestamp: string
} | null>(null)

async function fetchHealth() {
  loading.value = true
  error.value = null
  
  try {
    const response = await fetch('/api/health')
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }
    healthData.value = await response.json()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Unknown error'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchHealth()
})
</script>

<style scoped>
.container {
  max-width: 600px;
  margin: 2rem auto;
  padding: 1rem;
  font-family: system-ui, sans-serif;
}

h1 {
  color: #333;
}

.loading {
  color: #666;
}

.error {
  color: #dc2626;
  background: #fef2f2;
  padding: 1rem;
  border-radius: 8px;
}

.health-data {
  background: #f0fdf4;
  padding: 1rem;
  border-radius: 8px;
  border: 1px solid #86efac;
}

.health-data p {
  margin: 0.5rem 0;
}

button {
  margin-top: 1rem;
  padding: 0.5rem 1rem;
  background: #2563eb;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

button:hover {
  background: #1d4ed8;
}
</style>
