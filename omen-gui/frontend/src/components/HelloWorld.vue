<template>
  <main>
    <!-- generate the tabs header -->
    <div class="tab-container">
      <button 
        v-for="(tab, index) in tabs"
        :key="index" 
        :class="{ active: currentTab === index }" 
        @click="currentTab = index">
        {{ tab.name }}
      </button>
    </div>
    <!-- set main pane content depending on active tab -->
    <div class="tab-content">
      <div 
        v-for="(tab, index) in tabs" 
        :key="index" 
        class="tab-pane" 
        :class="{ active: currentTab === index }"
      >
        <APsTab v-if="index === 0" />
        <StationsTab v-if="index === 1"/>
        <NetsTab v-if="index === 2" />
      </div>
    </div>
    <div id="gen-result" class="result">{{ dynamic.gend }}</div>
    <div id="generate"> 
      <button class="btn" @click="generateJSON">Generate</button>
    </div>
  </main>
</template>

<script lang="ts" setup>
import { reactive, ref } from 'vue'
import {GenerateJSON} from '../../wailsjs/go/main/App'
import APsTab from './APsTab.vue'
import StationsTab from './StationsTab.vue'
import NetsTab from './NetsTab.vue'

// data that must update the UI automatically when changed/set
const dynamic = reactive({
  name: "",
  resultText: "Please enter your name below 👇",
  gend: "",
})

const currentTab = ref(0)

const tabs = [
  { name: 'APs' },
  { name: 'Stations' },
  { name: 'Nets' },
]


function generateJSON(){
    GenerateJSON().then(success => {
      if (success){dynamic.gend = "successfully generated input file"}
      else {dynamic.gend = "an error occurred"}
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
