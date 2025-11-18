<script lang="ts" setup>
import { main } from '../../wailsjs/go/models'
import { AddAP } from '../../wailsjs/go/main/App'
import { reactive, watch, ref } from 'vue'
import { GetNumberGroup } from "./shared.vue"

// define events
const emit = defineEmits<{
  valid: [is: boolean] // is currently valid?
}>()

//#region variables -----------------------------------------------------------

// NOTE: x, y, and z are composed into main.AP.position
let cur = ref(new main.AP({
  mode: main.WifiMode.a
}))
let c = reactive({
  x: 0,
  y: 0,
  z: 0 
})
// controlled by the following watch
let errors = ref(Array<string>())
// whenever a field (that requires validation) changes,
// check all fields for errors,
// emit whether or not we are in a valid state,
// and print error messages.
watch([() => cur.value.id, () => cur.value.position, () => c], () => {
  errors.value = validateAll()
  emit('valid', errors.value.length===0)
})

// list of APs already added to be displayed below as clickable buttons
const addedAPs = reactive(Array<string>())

//#endregion variables --------------------------------------------------------

function addAP() {
  // coalesce x,y,z into cur
  cur.value.position = `(${c.x},${c.y},${c.z})`

  // save off ID for later retrieval 
  addedAPs.push(cur.value.id)
  AddAP(cur.value)

  // reset the form for the next entry
  cur.value = new main.AP()

  c.x = 0
  c.y = 0
  c.z = 0
}

// validate checks all fields and returns a list of errors
function validateAll(): string[] {
  const msgs: string[] = []

  // test id
  if (cur.value.id.trim() == "")     msgs.push('ID is required')
  else { // populated-only tests
    if (!GetNumberGroup(cur.value.id)) msgs.push('ID must have exactly one number group')
    if (addedAPs.findIndex((v) => cur.value.id === v) != -1) msgs.push('AP ids must be unique')
  }
  // TODO additional rules
  return msgs
}
</script>

<template>
  <div>
    <input 
    v-model="cur.id"
    type="text"
    placeholder="ID" />
    <select v-model="cur.mode">
      <option v-for="mode in main.WifiMode">{{ mode }}</option>
    </select>
    <input v-model="cur.channel" type="number" placeholder="Channel" />
    <input v-model="cur.ssid" type="text" placeholder="SSID" />
    <br /><br />

    <div class="position-table">
      <div class="header-row">Position</div>
      <div class="row">
        <div class="cell"><label>X</label><input v-model="c.x" type="number" /></div>
        <div class="cell"><label>Y</label><input v-model="c.y" type="number" /></div>
        <div class="cell"><label>Z</label><input v-model="c.z" type="number" /></div>
      </div>
    </div>
    <br /><br />
    <button @click="addAP" :hidden="errors.length>0">Add AP</button>

    <!-- bubbles showing added AP IDs -->
    <div class="bubbles">
      <button v-for="id in addedAPs" :key="id" class="bubble" type="button" @click.stop>
        <!-- TODO click a button to load that AP's information-->
        {{ id }}
      </button>
    </div>
    <!-- errors -->
    <div class="error-list"> <!-- TODO define error-list class in css-->
      <div v-for="(err, idx) in errors"
        :key="idx"
      >{{ err }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.position-table {
  display: inline-block;
  border: 1px solid #ccc;
  padding: 10px;
}

.row {
  display: flex;
}

.cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin: 10px;
}

.cell label {
  font-weight: bold;
  margin-bottom: 5px;
}

.error-text label {
  color: rgb(252, 107, 107);
}

/* --- bubble styling --- */
.bubbles {
  margin-top: 12px;
  flex-wrap: wrap;
  gap: 3px;
  /* space between bubbles */
}

.bubble {
  background: #007aff;
  /* TODO replace with secondary color var */
  color: #fff;
  /* TODO replace with text color var */
  border: none;
  border-radius: 50%;
  /* rounded */
  justify-content: center;

  /* minimums so small bubbles look like circles */
  min-width: 2em;
  min-height: 2em;

  font-size: 12px;
}
</style>