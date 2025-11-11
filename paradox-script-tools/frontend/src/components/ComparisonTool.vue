<template>
  <div class="flex-1 flex flex-col p-6 overflow-auto bg-dark-input">
    <div class="bg-dark-panel/60 backdrop-blur-sm rounded-xl p-6 shadow-material border border-dark-border/50 mb-6">
      <h2 class="text-xl font-semibold mb-5">Comparison Tool</h2>

      <!-- FileSet A -->
      <div class="mb-4">
        <label class="block mb-2 font-medium">Compare FileSet/Directory A:</label>
        <textarea :value="setA.join('\n')" @input="setA = $event.target.value.split('\n').filter(p => p.trim())"
          rows="3" placeholder="Select files or directories..."
          class="w-full px-3 py-2 bg-dark-input/80 border border-dark-border rounded-lg text-gray-200 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-btn-primary/50 focus:border-btn-primary font-mono text-sm transition-all duration-200">
        </textarea>
        <div class="flex gap-2 mt-2">
          <button @click="selectFiles('Select Multiple Files To Compare (A)', '*.txt', 'setA')"
            class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
            Select File(s)
          </button>
          <button @click="selectFolder('Select Folder To Compare (A)', 'setA')"
            class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
            Select Folder
          </button>
          <button @click="clearSet('setA')"
            class="px-4 py-2 bg-gray-600/80 hover:bg-gray-700/80 text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
            Clear
          </button>
        </div>
      </div>

      <!-- FileSet B -->
      <div class="mb-4">
        <label class="block mb-2 font-medium">Compare FileSet/Directory B:</label>
        <textarea :value="setB.join('\n')" @input="setB = $event.target.value.split('\n').filter(p => p.trim())"
          rows="3" placeholder="Select files or directories..."
          class="w-full px-3 py-2 bg-dark-input/80 border border-dark-border rounded-lg text-gray-200 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-btn-primary/50 focus:border-btn-primary font-mono text-sm transition-all duration-200">
        </textarea>
        <div class="flex gap-2 mt-2">
          <button @click="selectFiles('Select Multiple Files To Compare (B)', '*.txt', 'setB')"
            class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
            Select File(s)
          </button>
          <button @click="selectFolder('Select Folder To Compare (B)', 'setB')"
            class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
            Select Folder
          </button>
          <button @click="clearSet('setB')"
            class="px-4 py-2 bg-gray-600/80 hover:bg-gray-700/80 text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
            Clear
          </button>
        </div>
      </div>

      <!-- Compare Button -->
      <button @click="updateMatchingFiles" :disabled="loadingFiles || setA.length === 0 || setB.length === 0"
        class="w-full mt-4 px-4 py-3 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg font-medium transition-all duration-200 shadow-material hover:shadow-material-lg disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-material">
        {{ loadingFiles ? 'Finding matching files for comparison...' : 'Compare' }}
      </button>
    </div>

    <!-- File List -->
    <div v-if="matchingFiles.length > 0"
      class="flex-1 bg-dark-panel/60 backdrop-blur-sm rounded-xl p-6 shadow-material border border-dark-border/50 overflow-hidden flex flex-col">
      <h3 class="text-lg font-semibold mb-4">Matching Files ({{ matchingFiles.length }})</h3>
      <div class="flex-1 overflow-y-auto">
        <div v-for="(file, index) in matchingFiles" :key="index" @click="viewFileDiff(file)"
          class="p-3 mb-2 bg-dark-input/60 rounded-lg border border-dark-border/30 shadow-material cursor-pointer hover:bg-dark-input/80 transition-all duration-200">
          <div class="text-sm font-mono text-gray-200">{{ file.relativePath }}</div>
        </div>
      </div>
    </div>
    <div v-else-if="!loadingFiles"
      class="bg-dark-panel/60 backdrop-blur-sm rounded-xl p-6 shadow-material border border-dark-border/50">
      <p class="text-gray-400 text-center">
        Please select both sets of files/directories and click "Compare"
      </p>
    </div>

    <!-- Diff Viewer Overlay -->
    <DiffViewer :visible="diffFilePath !== null" :lines="diffLines" :loading="loadingDiff" @close="closeDiff" />
  </div>
</template>

<script>
import { SelectDirectory, SelectMultipleFiles, CollectFilesFromPaths, FindMatchingFiles, GetDiff } from '../../wailsjs/go/main/App'
import DiffViewer from './DiffViewer.vue'

export default {
  name: 'ComparisonTool',
  emits: ['view-report'],
  components: {
    DiffViewer
  },
  data() {
    return {
      setA: [],
      setB: [],
      matchingFiles: [],
      loadingFiles: false,
      diffLines: [],
      diffFilePath: null,
      loadingDiff: false
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
      this.matchingFiles = []
    },
    async updateMatchingFiles() {
      if (this.setA.length === 0 || this.setB.length === 0) {
        alert('Please select at least one file or directory for both sets')
        this.matchingFiles = []
        return
      }

      this.loadingFiles = true
      try {
        // Collect files from both sets
        const filesA = await CollectFilesFromPaths(this.setA)
        const filesB = await CollectFilesFromPaths(this.setB)

        // Find matching files
        const matches = await FindMatchingFiles(filesA, filesB)

        // Convert to array format for display
        this.matchingFiles = Object.keys(matches).map(relativePath => ({
          relativePath,
          fileAPath: matches[relativePath].FileAPath,
          fileBPath: matches[relativePath].FileBPath
        }))
      } catch (error) {
        alert('Error finding matching files: ' + error)
        this.matchingFiles = []
      } finally {
        this.loadingFiles = false
      }
    },
    async viewFileDiff(file) {
      this.loadingDiff = true
      this.diffFilePath = file.relativePath

      try {
        const diffLines = await GetDiff(file.fileAPath, file.fileBPath)
        this.diffLines = diffLines || []
      } catch (error) {
        alert('Error loading diff: ' + error)
        this.diffLines = []
      } finally {
        this.loadingDiff = false
      }
    },
    closeDiff() {
      this.diffLines = []
      this.diffFilePath = null
      this.loadingDiff = false
    },
    async viewReport() {
      // Emit event to parent if needed
      this.$emit('view-report', null)
    }
  }
}
</script>
