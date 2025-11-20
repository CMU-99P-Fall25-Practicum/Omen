<template>
  <main>
    <!-- main tab content -->
    <div v-if="currentTab === 'main'">
      <h1 class="section-header">SSH Connection</h1>
      <div>
        <label class="field">Username</label>: <input v-model="sections.main.username" type="text">
        <label class="field">Password</label>: <input v-model="password" type="password">
      </div>
      <div>
        <label class="field">Host</label>: <input v-model="host" type="text"> <label>Port</label>:
        <input v-model="port" type="number" min="1" max="65535">
      </div>
      <div class="error-list">
        <div v-for="(err, idx) in validationErrors" :key="idx">{{ err }}</div>
      </div>
      <hr />
      <h1 class="section-header">Wireless Propagation Settings</h1>
      <div>
        <label class="field">Noise Threshold</label>:
        <input title="Noise Threshold sets the value (in dBm) below which a message is considered lost." v-model="noise_threshold" type="number" placeholder="Noise Threshold">
        <br />
        <h2>Model</h2>
        <select v-model="model.m">
          <option v-for="name in main.PropModel">{{ name }}</option>
        </select>
        <br />
        <label class="field">Exponent</label>:
        <input v-model="model.exp" type="number" placeholder="Exponent">
        <br />
        <label class="field">Standard Deviation</label>:
        <input v-model="model.s" type="number" placeholder="S">
      </div>
    </div>
    <hr />
    <!-- the other two tabs are pulled from child files -->
    <h1>Access Points</h1>
    <APsTab @APsCount="APsValid" />
    <hr />
    <h1>Stations</h1>
    <StationsTab @stationsChanged="StationsValid" />
    <hr />
    <div id="generate">
      <!-- this button is only enabled if every tab has self-reported as valid-->
      <button class="generate-button" v-show="sections.APs.valid && sections.Stations.valid && sections.main.valid"
        @click="generateJSON">Generate</button>
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

// variables used by this tab
const password = ref(''),
  host = ref('127.0.0.1'),
  port = ref(22),
  noise_threshold = ref(-100),
  model = reactive({
    m: main.PropModel.LogNormalShadowing, exp: 0,
    s: 0
  }),
  generation_result = ref(''), // result of the last GenerateJSON call
  currentTab = ref('main') // which tab is currently active and displaying

// #region tab handling and validation ----------------------------------------

// validity state of the sections.
// Sections with sub-documents are self-contained and thus only need a valid bool.
const sections = reactive({
  main: { valid: false, username: '' }, // this tab
  APs: { valid: false },
  Stations: { valid: false }
})

function APsValid(count: number) {
  sections['APs'].valid = (count > 0)
  console.warn(['APs tab is valid:', sections['APs'].valid])
}

function StationsValid(count: number) {
  sections['Stations'].valid = (count > 0)
  console.warn(['Station tab is valid:', sections['Stations'].valid])
}

// #endregion tab handling ----------------------------------------------------

// check each field for validation errors whenever one changes
let validationErrors = computed(() => {
  const msgs: string[] = []

  if (sections.main.username.trim() === '') msgs.push('SSH username cannot be empty')
  if (host.value.trim() === '') msgs.push('SSH host cannot be empty')
  else {
    // populated-ony checks
    if (!IsValidIP(host.value)) msgs.push('SSH host must be a valid IPv4 or IPv6 address')
  }
  if (port.value < 1 || port.value > (2 << 16) - 1) msgs.push('Port must be between 1 and 65535')

  sections.main.valid = (msgs.length === 0)
  console.warn(['main tab is valid: ', sections.main.valid])

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

<style>

.result {
  height: 20px;
  line-height: 20px;
  margin: 1.5rem auto;
}
</style>
