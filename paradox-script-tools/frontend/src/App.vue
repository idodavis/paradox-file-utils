<template>
  <div class="flex flex-col h-screen bg-dark-bg text-gray-200">
    <header
      class="bg-dark-panel/80 backdrop-blur-md px-8 py-4 border-b border-dark-border flex justify-between items-center shadow-material z-50 relative">
      <h1 class="text-2xl font-semibold tracking-tight">Paradox Script Tools</h1>
      <nav class="flex gap-3">
        <button @click="showComparisonTool = false" :class="[
          'px-5 py-2.5 rounded-lg border transition-all duration-200 font-medium',
          !showComparisonTool
            ? 'bg-btn-primary border-btn-primary text-white shadow-material hover:shadow-material-lg'
            : 'bg-transparent border-dark-border text-gray-400 hover:border-gray-500 hover:text-gray-300'
        ]">
          Merge Files
        </button>
        <button @click="showComparisonTool = true" :class="[
          'px-5 py-2.5 rounded-lg border transition-all duration-200 font-medium',
          showComparisonTool
            ? 'bg-btn-primary border-btn-primary text-white shadow-material hover:shadow-material-lg'
            : 'bg-transparent border-dark-border text-gray-400 hover:border-gray-500 hover:text-gray-300'
        ]">
          Comparison Tool
        </button>
      </nav>
    </header>

    <main class="flex-1">
      <MergeTool v-show="!showComparisonTool" />
      <ComparisonTool v-show="showComparisonTool" @view-report="viewCompToolReport" />
    </main>
  </div>
</template>

<script>
import MergeTool from './components/MergeTool.vue'
import ComparisonTool from './components/ComparisonTool.vue'

export default {
  name: 'App',
  components: {
    MergeTool,
    ComparisonTool
  },
  data() {
    return {
      showComparisonTool: false
    }
  },
  methods: {
    async viewCompToolReport(comparisonOutput) {
      if (!comparisonOutput) {
        return
      }

      try {
        // TODO: View Comparison Report Logic
      } catch (error) {
        alert('Error generating comparison report: ' + error)
      }
    }
  }
}
</script>
