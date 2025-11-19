<template>
  <main>
    <!-- generate the tabs header -->
    <div class="tab-container">
      <button
        v-for="(_, key) in tabs"
        :key="key"
        :class="{ active: currentTab === key }"
        @click="currentTab = key">
        {{ key }}
      </button>
    </div>
    <!-- set main pane content depending on active tab -->
    <div class="tab-content">
      <div
        v-for="(_, key) in tabs"
        :key="key"
        class="tab-pane"
        :class="{ active: currentTab === key }">
        <MainTab
          @valid="(v) => (tabs['main'].valid = v)"
          v-if="currentTab === 'main'" />
        <APsTab
          @valid="(v) => (tabs['APs'].valid = v)"
          v-if="currentTab === 'APs'" />
        <StationsTab @stationsChanged="(count) => (tabs['Stations'].valid = count>0)" v-if="currentTab === 'Stations'" />
        <NetsTab v-if="currentTab === 'Nets'" />
      </div>
    </div>
    <div id="generate">
      <!-- this button is only enabled if every tab has self-reported as valid-->
      <button class="btn" :disabled="!allTabsValid" @click="generateJSON">Generate</button>
      <div id="generate-result" class="result">{{ dynamic.gend }}</div>
    </div>
  </main>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from 'vue'
import { GenerateJSON } from '../wailsjs/go/main/App'
import APsTab from './components/APsTab.vue'
import StationsTab from './components/StationsTab.vue'
import NetsTab from './components/NetsTab.vue'
import MainTab from './components/MainTab.vue'

// are all tabs in a valid state?
const allTabsValid = computed(() => Object.values(tabs).every((v) => v.valid))

// data that must update the UI automatically when changed/set
const dynamic = reactive({ name: '', gend: '' })

const currentTab = ref('main')

const tabs = {
  main: { valid: false },
  APs: { valid: false },
  Stations: { valid: false },
  Nets: { valid: false },
}

function generateJSON() {
  GenerateJSON().then((success) => {
    if (success) {
      dynamic.gend = 'successfully generated input file'
    } else {
      dynamic.gend = 'an error occurred'
    }
  })
}
</script>

<style scoped>
.tab-container {
  display: flex;
  border-bottom: 1px solid #ccc;
}

.tab-container button {
  padding: 10px 20px;
  border: none;
  border-radius: 5px 5px 0 0;
  cursor: pointer;
}

.tab-container button.active {
  background-color: #ccc;
}

.tab-content {
  padding: 20px;
}

.tab-pane {
  display: none;
}

.tab-pane.active {
  display: block;
}

.result {
  height: 20px;
  line-height: 20px;
  margin: 1.5rem auto;
}

.btn {
  width: 60px;
  height: 30px;
  line-height: 30px;
  border-radius: 3px;
  border: none;
  margin: 0 0 0 20px;
  padding: 0 8px;
  cursor: pointer;
}
</style>
