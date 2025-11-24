<script lang="ts" setup>
import { main } from '../../wailsjs/go/models'
import { AddAP } from '../../wailsjs/go/main/App'
import { reactive, computed, watchEffect } from 'vue'
import { CoalescePosition, GetNumberGroup } from './shared.vue'

const emit = defineEmits<{
  APsCount: [count: number] // the number APs that will be generated
}>()

// list of APs already added to be displayed below as clickable buttons
const addedAPs = reactive(Array<string>()),
  curAP = reactive(new main.AP({
    id: 'ap1',
    mode: main.WifiMode.a,
    channel: 0,
    ssid: '',
    position: '' // NOTE: x, y, and z are composed into main.AP.position
  })),
  pos = reactive({ x: 0, y: 0, z: 0 })
// validation errors is recomputed every time cur (as its one dependency) is touched.
// It is used to disable the Add AP button and provide reasons why.
let validationErrors = computed(() => {
  const msgs: string[] = []

  // test id
  if (curAP.id.trim() == '') msgs.push('ID is required')
  else {
    // populated-only tests
    {
      let ng: string = GetNumberGroup(curAP.id)
      if (ng == '') msgs.push('ID must have exactly one number group')
      if (Number(ng) < 0) msgs.push('ID number group must be positive')
    }
    if (addedAPs.findIndex((v) => curAP.id === v) != -1) msgs.push('AP ids must be unique')
  }

  return msgs
})

// alert our parent about our current state
watchEffect(() => { emit('APsCount', addedAPs.length) })

// addAP passes the current AP information to the backend and clears out the existing data.
function addAP() {
  // coalesce x,y,z into cur
  curAP.position = CoalescePosition(pos.x, pos.y, pos.z)

  // save off ID for later retrieval
  addedAPs.push(curAP.id)
  AddAP(curAP)

  // determine default values for next AP
  let newID: number = Number(GetNumberGroup(curAP.id)) + 1

  // reset the form for the next entry
  curAP.id = 'ap' + String(newID)
  curAP.mode = main.WifiMode.a
  curAP.channel = 0
  // do not touch ssid
  curAP.position = ''

  pos.x = 0
  pos.y = 0
  pos.z = 0
}
</script>

<template>
  <div>
    <label class='field'>ID</label>:
    <input v-model="curAP.id" type="text" placeholder="ID">
    <br />
    <label class='field'>Wifi Mode</label>:
    <select v-model="curAP.mode">
      <option v-for="mode in main.WifiMode">{{ mode }}</option>
    </select>
    <br />
    <label class='field'>Channel</label>:
    <input v-model="curAP.channel" type="number" placeholder="Channel" min="0">
    <br />
    <label class='field'>SSID</label>:
    <input v-model="curAP.ssid" type="text" placeholder="SSID">
    <br />
    <div class="position-table">
      <div class="header-row">Position</div>
      <div class="row">
        <div class="cell"><label>X</label><input v-model="pos.x" type="number"></div>
        <div class="cell"><label>Y</label><input v-model="pos.y" type="number"></div>
        <div class="cell"><label>Z</label><input v-model="pos.z" type="number"></div>
      </div>
    </div>
    <br><br>
    <!-- bubbles showing added AP IDs -->
    <div class="bubbles">
      <button v-for="id in addedAPs" :key="id" class="bubble" type="button" @click.stop>
        <!-- TODO click a button to load that AP's information-->
        {{ id }}
      </button>
    </div>
    <button class="add-button" @click="addAP" v-show="validationErrors.length === 0">Add AP</button>
    <div class="error-text">
      <div v-for="(err, idx) in validationErrors" :key="idx">{{ err }}</div>
    </div>
    <div class="error-text">
      <div v-show="addedAPs.length===0">You must add at least 1 access point.</div>
    </div>
  </div>
</template>