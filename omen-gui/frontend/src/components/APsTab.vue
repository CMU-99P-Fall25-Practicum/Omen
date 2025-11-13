<script lang="ts" variables>

</script>
 

<template>
  <div>
      <input v-model="cur.id" type="text" placeholder="ID" />
      <select v-model="cur.mode">
        <option value="a">a</option>
        <option value="b">b</option>
      </select>
      <input v-model="cur.channel" type="text" placeholder="Channel" />
      <input v-model="cur.ssid" type="text" placeholder="SSID" />
      <input v-model="cur.position" type="text" placeholder="Position" />
    <button @click="addAP">Add AP</button>
  </div>
</template>

<script lang="ts" setup>
import { main } from '../../wailsjs/go/models'
import { AddAP } from '../../wailsjs/go/main/App'

var cur = new main.AP()

function addAP() {
  AddAP(cur)
}
</script>