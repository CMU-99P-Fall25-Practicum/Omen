<script lang="ts" setup>
import { main } from '../../wailsjs/go/models'
import { reactive, computed, watchEffect } from 'vue'
import { AddSta } from '../../wailsjs/go/main/App'
import { GetNumberGroup } from './shared.vue'

const emit = defineEmits<{
  stationsChanged: [count: number] // the number of stations that will be generated
}>()

/*export*/ const AddedStas = reactive(Array<string>())
const curSta = reactive(
  new main.Sta({
    id: 'sta1',
    position: ''
  })
),
  validationErrors = computed(() => {
    const msgs: string[] = []

    // test id
    if (curSta.id.trim() == '') msgs.push('ID is required')
    else {
      // populated-only tests
      {
        let ng: string = GetNumberGroup(curSta.id)
        if (ng == '') msgs.push('ID must have exactly one number group')
        if (Number(ng) < 0) msgs.push('ID number group must be positive')
      }
      if (AddedStas.findIndex((v) => curSta.id === v) != -1)
        msgs.push('station ids must be unique')
    }

    return msgs
  })
const pos = reactive({ x: 0, y: 0, z: 0 })

// alert our parent whenever a station is added or removed
watchEffect(() => {
  emit('stationsChanged', AddedStas.length)
})

function addStation() {
  curSta.position = `(${pos.x},${pos.y},${pos.z})`

  AddedStas.push(curSta.id)
  // pass to the backend
  AddSta(curSta)

  // determine default values for next station
  let newID: number = Number(GetNumberGroup(curSta.id)) + 1

  // reset the form for the next entry
  curSta.id = 'sta' + String(newID)
  curSta.position = ''

  pos.x = 0
  pos.y = 0
  pos.z = 0
}
</script>

<template>
  <div>
    <label class='field'>ID</label>: <input v-model="curSta.id" type="text" placeholder="staX">
    <br />
    <div class="position-table">
      <div class="header-row">Position</div>
      <div class="row">
        <div class="cell"><label>X</label><input v-model="pos.x" type="number"></div>
        <div class="cell"><label>Y</label><input v-model="pos.y" type="number"></div>
        <div class="cell"><label>Z</label><input v-model="pos.z" type="number"></div>
      </div>
    </div>
    <button @click="addStation" v-show="validationErrors.length === 0">Add Station</button>
    <div class="error-list">
      <div v-for="(err, idx) in validationErrors" :key="idx">{{ err }}</div>
    </div>
  </div>
</template>
