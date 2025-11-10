<template>
  <div v-if="visible"
    class="col-span-2 bg-dark-panel/60 backdrop-blur-sm rounded-xl p-6 shadow-material border border-dark-border/50">
    <h2 class="text-xl font-semibold mb-5">Comparison Tool</h2>

    <!-- FileSet A -->
    <div class="mb-4">
      <label class="block mb-2 font-medium">Compare FileSet/Directory A:</label>
      <textarea :value="setA.join('\n')" @input="setA = $event.target.value.split('\n').filter(p => p.trim())" rows="3"
        placeholder="Select files or directories..."
        class="w-full px-3 py-2 bg-dark-input/80 border border-dark-border rounded-lg text-gray-200 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-btn-primary/50 focus:border-btn-primary font-mono text-sm transition-all duration-200">
      </textarea>
      <div class="flex gap-2 mt-2">
        <button @click="selectFiles('A', '*.txt', 'setA')"
          class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
          Select File(s)
        </button>
        <button @click="selectFolder('A', 'setA')"
          class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
          Select Folder
        </button>
        <button @click="clearSet('A')"
          class="px-4 py-2 bg-gray-600/80 hover:bg-gray-700/80 text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
          Clear
        </button>
      </div>
    </div>

    <!-- FileSet B -->
    <div class="mb-4">
      <label class="block mb-2 font-medium">Compare FileSet/Directory B:</label>
      <textarea :value="setB.join('\n')" @input="setB = $event.target.value.split('\n').filter(p => p.trim())" rows="3"
        placeholder="Select files or directories..."
        class="w-full px-3 py-2 bg-dark-input/80 border border-dark-border rounded-lg text-gray-200 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-btn-primary/50 focus:border-btn-primary font-mono text-sm transition-all duration-200">
      </textarea>
      <div class="flex gap-2 mt-2">
        <button @click="selectFiles('B', '*.txt', 'setB')"
          class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
          Select File(s)
        </button>
        <button @click="selectFolder('B', 'setB')"
          class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
          Select Folder
        </button>
        <button @click="clearSet('B')"
          class="px-4 py-2 bg-gray-600/80 hover:bg-gray-700/80 text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
          Clear
        </button>
      </div>
    </div>

    <!-- Run Comparison Button -->
    <button @click="runComparison" :disabled="comparing"
      class="w-full px-4 py-3 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg font-medium transition-all duration-200 shadow-material hover:shadow-material-lg disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-material">
      {{ comparing ? 'Comparing...' : 'Run Comparison' }}
    </button>

    <!-- Results -->
    <div v-if="comparisonOutput"
      class="mt-4 p-4 bg-dark-input/60 rounded-lg border border-dark-border/30 shadow-material">
      <h3 class="text-lg font-semibold mb-2">Results</h3>
      <p class="mb-1">Changed: {{ comparisonOutput.ChangedFiles?.length || 0 }}</p>
      <p class="mb-1">Added: {{ comparisonOutput.AddedFiles?.length || 0 }}</p>
      <p class="mb-4">Removed: {{ comparisonOutput.RemovedFiles?.length || 0 }}</p>
      <button @click="viewReport"
        class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
        View Full Report
      </button>
    </div>
  </div>
</template>

<script>
import { SelectDirectory, SelectMultipleFiles } from '../../wailsjs/go/main/App'

export default {
  name: 'ComparisonTool',
  props: {
    visible: {
      type: Boolean,
      default: false
    }
  },
  emits: ['view-report'],
  data() {
    return {
      setA: [],
      setB: [],
      comparing: false,
      comparisonOutput: null
    }
  },
  methods: {
    async selectFiles(dialogTitle, filter, fileSet) {
      try {
        const selected = await SelectMultipleFiles(dialogTitle, filter)
        if (selected && selected.length > 0) {
          // Add new files, avoiding duplicates
          const existing = new Set(this[fileSet])
          for (const file of selected) {
            if (!existing.has(file)) {
              this[fileSet].push(file)
            }
          }
        }
      } catch (error) {
        alert('Error selecting files: ' + error)
      }
    },
    async selectFolder(dialogTitle, fileSet) {
      try {
        const selected = await SelectDirectory(dialogTitle)
        if (selected) {
          this[fileSet].push(selected)
        }
      } catch (error) {
        alert('Error selecting folder: ' + error)
      }
    },
    clearSet(fileSet) {
      this[fileSet] = []
    },
    async runComparison() {
      this.comparing = true
      this.comparisonOutput = null

      try {
        // TODO: Comparison Tool Logic
      } catch (error) {
        alert('Error comparing directories: ' + error)
      } finally {
        this.comparing = false
      }
    },
    async viewReport() {
      if (!this.comparisonOutput) {
        return
      }

      try {
        // TODO: View Comparison Report Logic
        // Emit event to parent to show report in diff viewer
        this.$emit('view-report', this.comparisonOutput)
      } catch (error) {
        alert('Error generating comparison report: ' + error)
      }
    }
  }
}
</script>
