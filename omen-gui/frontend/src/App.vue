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
        <div v-if="currentTab === 'main'">
          <div>
    <label>Username</label>: <input v-model="username" type="text"> <label>Password</label>:
    <input v-model="password" type="password">
  </div>
  <div>
    <label>Host</label>: <input v-model="host" type="text"> <label>Port</label>:
    <input v-model="port"
      type="number"
      min="1"
      max="65535">
  </div>
  <hr />
  <h1 class="section-header">Wireless Propagation Settings</h1>
  <div>
    <input v-model="noise_threshold" type="number" placeholder="Noise Threshold">
    <select v-model="model.m">
      <option v-for="name in main.PropModel">{{ name }}</option>
    </select>
    <input v-model="model.exp" type="number" placeholder="Exponent">
    <input v-model="model.s" type="number" placeholder="S">
  </div>
  <div class="error-list">
    <div v-for="(err, idx) in validationErrors" :key="idx">{{ err }}</div>
  </div>
        </div>
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
      <div id="generate-result" class="result">{{ generation_result }}</div>
    </div>
  </main>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from 'vue'
import { GenerateJSON } from '../wailsjs/go/main/App'
import APsTab from './components/APsTab.vue'
import StationsTab from './components/StationsTab.vue'
import { main } from '../wailsjs/go/models'
import { isValid as IsValidIP } from 'ipaddr.js'

// #region tab handling -------------------------------------------------------

const allTabsValid  = computed(() => Object.values(tabs).every((v) => v.valid)),
  generation_result = ref(''),
  currentTab        = ref('main')

const tabs = {
  main: { valid: false }, // this tab
  APs: { valid: false },
  Stations: { valid: false },
}

// #endregion tab handling ----------------------------------------------------

  const username = ref(''),
    password = ref(''),
    host = ref('127.0.0.1'),
    port = ref(22),
    noise_threshold = ref(-100),
    model = reactive({
      m: main.PropModel.LogNormalShadowing,
      exp: 0,
      s: 0})

  // check each field for validation errors whenever one changes
  let validationErrors = computed(() => {
    const msgs: string[] = []

    if (username.value.trim() === '') msgs.push('SSH username cannot be empty')
    if (host.value.trim() === '') msgs.push('SSH host cannot be empty')
    else {
      // populated-ony checks
      if (!IsValidIP(host.value)) msgs.push('SSH host must be a valid IPv4 or IPv6 address')
    }
    if (port.value < 1 || port.value > (2 << 16) - 1) msgs.push('Port must be between 1 and 65535')

    return msgs
  })

// generateJSON invokes the backend to create an input.json file.
// Success or failure is placed in a local variable for display.
function generateJSON() {
  GenerateJSON().then((success) => {
    if (success) {
      generation_result.value = 'successfully generated input file'
    } else {
      generation_result.value = 'an error occurred'
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
