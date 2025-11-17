<template>
  <div>
    <input v-model="cur.id" type="text" placeholder="ID" />
    <!-- TODO validate a unique number exists across stations and aps -->
    <select v-model="cur.mode">
      <option value={{main.WifiMode.a}}>a</option>
      <option value={{main.WifiMode.b}}>b</option>
    </select>
    <input v-model="cur.channel" type="number" placeholder="Channel" />
    <input v-model="cur.ssid" type="text" placeholder="SSID" />
  <br/><br/>

    <div class="position-table">
      <div class="header-row">Position</div>
      <div class="row">
        <div class="cell"><label>X</label><input v-model="x" type="number" /></div>
        <div class="cell"><label>Y</label><input v-model="y" type="number" /></div>
        <div class="cell"><label>Z</label><input v-model="z" type="number" /></div>
      </div>
      <div class="error-text">
        <label>{{errors.position}}</label>
      </div>
    </div>
    <button @click="addAP">Add AP</button>

    <!-- bubbles showing added AP IDs -->
    <div class="bubbles">
      <button
      v-for="id in addedAPs"
        :key="id"
        class="bubble"
        type="button"
        @click.stop
      > <!-- TODO click a button to load that AP's information-->
        {{ id }}
      </button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { main } from '../../wailsjs/go/models'
import { AddAP } from '../../wailsjs/go/main/App'
import { reactive } from 'vue'

// NOTE: x, y, and z are composed into main.AP.position
let cur = new main.AP()
let x: number = 0,
    y: number = 0,
    z: number = 0

var errors = {
 position: 'test',
}

// list of APs already added to be displayed below as clickable buttons
const addedAPs = reactive(Array<string>())

function addAP() {
  // coalesce x,y,z into cur
  cur.position = `(${x},${y},${z})`

  // save off ID for later retrieval 
  addedAPs.push(cur.id)
  AddAP(cur)

  // reset the form for the next entry
  cur = new main.AP()
  x = y = z = 0
}
</script>

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
  gap: 3px; /* space between bubbles */
}
.bubble {
  background: #007aff; /* TODO replace with secondary color var */
  color: #fff; /* TODO replace with text color var */
  border: none;
  border-radius: 50%; /* rounded */
  justify-content: center;

  /* minimums so small bubbles look like circles */
  min-width: 2em;
  min-height: 2em;

  font-size: 12px;
}
</style>