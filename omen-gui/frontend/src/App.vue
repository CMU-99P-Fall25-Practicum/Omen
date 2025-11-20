<template>
  <main>
    <!-- main tab content -->
    <div>
      <h1 class="section-header">SSH Connection</h1>
      <div>
        <label class="field">Username</label>: <input v-model="sections.main.username" type="text">
        <label class="field">Password</label>: <input v-model="sections.main.password" type="password">
      </div>
      <div>
        <label class="field">Host</label>: <input v-model="sections.main.host" type="text"> <label>Port</label>:
        <input v-model="sections.main.port" type="number" min="1" max="65535">
      </div>
      <div class="error-list">
        <div v-for="(err, idx) in validationErrors" :key="idx">{{ err }}</div>
      </div>
      <hr />
      <h1 class="section-header">Wireless Propagation Settings</h1>
      <div>
        <label class="field">Noise Threshold</label>:
        <input title="Noise Threshold sets the value (in dBm) below which a message is considered lost."
          v-model="sections.main.nets.noise_th" type="number" placeholder="Noise Threshold">
        <br />
        <h2>Model</h2>
        <select v-model="sections.main.nets.propagation_model.model">
          <option v-for="name in main.PropModel">{{ name }}</option>
        </select>
        <br />
        <label class="field">Exponent</label>:
        <input v-model="sections.main.nets.propagation_model.exp" type="number" placeholder="Exponent">
        <br />
        <label class="field">Standard Deviation</label>:
        <input v-model="sections.main.nets.propagation_model.s" type="number" placeholder="S">
      </div>
    </div>
    <hr />
    <!-- the other two tabs are pulled from child files -->
    <h1 class="section-header">Access Points</h1>
    <APsTab @APsCount="APsValid" />
    <hr />
    <h1 class="section-header">Stations</h1>
    <StationsTab @stationsChanged="StationsValid" />
    <hr />
    <div>
      <h1 class="section-header">Tests</h1>
      <!-- display each timeframe and enable the creation of more -->
      <button @click="addTimeframe" class="add-timeframe">Add timeframe</button>
      <div v-for="(tf, tfIdx) in sections.Timeframes" :key=tfIdx class="timeframe">
        <h2>Timeframe {{ tfIdx + 1 }}</h2>

        <!-- enable adding more tests within this timeframe -->
        <button @click="addTest(tfIdx)" class="add-test">Add test</button>
        <div v-for="(test, _) in tf.tests" class="test-row">
          <label class="field">Node:</label>
          <input v-model="test.node" type="text" placeholder="node" />

          <div class="position-table">
            <div class="header-row">Position</div>
            <div class="row">
              <div class="cell"><label>X</label><input v-model="test.x" type="number"></div>
              <div class="cell"><label>Y</label><input v-model="test.y" type="number"></div>
              <div class="cell"><label>Z</label><input v-model="test.z" type="number"></div>
            </div>
          </div>
        </div>
      </div>
      <hr />
      <div id="generate">
        <!-- this button is only enabled if every tab has self-reported as valid-->
        <button class="generate-button" v-show="sections.APs.valid && sections.Stations.valid && sections.main.valid"
          @click="generateJSON">Generate</button>
        <div id="generate-result" class="result">{{ generation_result }}</div>
      </div>
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
import { CoalescePosition } from './components/shared.vue'

// variables used by this tab
const generation_result = ref('') // result of the last GenerateJSON call

// #region tab handling and validation ----------------------------------------

// validity state of the sections.
// Sections with sub-documents are self-contained and thus only need a valid bool.
const sections = reactive({
  main: {
    valid: false,
    username: '',
    password: '',
    nets: new main.Nets({
      noise_th: -100,
      propagation_model: new main.PropagationModel({
        model: main.PropModel.LogNormalShadowing,
        exp: 0,
        s: 0,
      })
    }),
    host: '127.0.0.1',
    port: 22,
    tests: [new main.Test({})],
  }, // this tab
  APs: { valid: false },
  Stations: { valid: false },
  Timeframes: [{
    tests: [{
      node: '',
      x: 0,
      y: 0,
      z: 0
    }],
  }]
})

function APsValid(count: number) {
  sections['APs'].valid = (count > 0)
  console.warn(['APs tab is valid:', sections['APs'].valid])
}

function StationsValid(count: number) {
  sections['Stations'].valid = (count > 0)
  console.warn(['Station tab is valid:', sections['Stations'].valid])
}

// addTimeframe inserts a new timeframe into the local holder of timeframes.
function addTimeframe() {
  sections.Timeframes.push({
    tests: [{
      node: '',
      x: 0,
      y: 0,
      z: 0
    }],
  })
}

// addTest inserts a new test into the given timeframe.
function addTest(tfIdx: number) {
  const tf = sections.Timeframes[tfIdx]
  if (!tf) return

  tf.tests.push({
    node: '',
    x: 0,
    y: 0,
    z: 0
  })
}

// collapseTests coalesces timeframes and their tests into how the backend expects them.
function collapseTests(): Array<main.Test> {
  const result: Array<main.Test> = []

  sections.Timeframes.forEach((tf, tfIdx) => {
    tf.tests.forEach(test => {
      const pos = CoalescePosition(test.x, test.y, test.z)
      result.push(
        new main.Test({
          name: `move ${test.node} to ${pos}`,
          type: 'node movements',
          timeframe: tfIdx + 1, // our timeframes are 1-indexed
          node: test.node,
          position: pos,
        }))
    })
  })

  return result
}

// #endregion tab handling ----------------------------------------------------

// check each field for validation errors whenever one changes
let validationErrors = computed(() => {
  const msgs: string[] = []

  if (sections.main.username.trim() === '') msgs.push('SSH username cannot be empty')
  if (sections.main.host.trim() === '') msgs.push('SSH host cannot be empty')
  else {
    // populated-ony checks
    if (!IsValidIP(sections.main.host)) msgs.push('SSH host must be a valid IPv4 or IPv6 address')
  }
  if (sections.main.port < 1 || sections.main.port > (2 << 16) - 1) msgs.push('Port must be between 1 and 65535')

  sections.main.valid = (msgs.length === 0)
  console.warn(['main tab is valid: ', sections.main.valid])

  return msgs
})

// generateJSON invokes the backend to create an input.json file.
// Success or failure is placed in a local variable for display.
function generateJSON() {
  GenerateJSON('run_name',
    sections.main.username, sections.main.password,
    sections.main.host, sections.main.port,
    sections.main.nets, sections.main.tests).then((success) => {
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
