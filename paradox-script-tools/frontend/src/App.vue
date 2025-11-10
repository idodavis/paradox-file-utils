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

    <main class="flex-1 grid grid-cols-2 gap-6 p-6 overflow-auto bg-dark-input">
      <!-- Configuration Panel -->
      <div v-if="!showComparisonTool"
        class="bg-dark-panel/60 backdrop-blur-sm rounded-xl p-6 overflow-y-auto shadow-material border border-dark-border/50">
        <!-- Merge Options -->
        <div class="mb-6">
          <div class="mb-4">
            <h3 class="text-lg font-semibold mb-3">Merge Options</h3>
            <p class="text-sm text-gray-400 mb-5 leading-relaxed">File Set A (Base) entries take precedence in
              conflicts. To use B as
              base, swap the directories.</p>

            <div class="flex items-center cursor-pointer ml-4">
              <input type="checkbox" v-model="addAdditionalEntries"
                class="w-4 h-4 text-btn-primary bg-dark-input border-dark-border rounded focus:ring-btn-primary focus:ring-2" />
              <span class="ml-2">
                Add additional entries from B that are not in A
              </span>
            </div>

            <div v-if="addAdditionalEntries" class="mb-4 ml-8">
              <div class="my-2 text-sm text-gray-400">Additional Entry Placement:</div>
              <div class="space-y-2">
                <label class="flex items-center cursor-pointer">
                  <input type="radio" v-model="entryPlacement" value="bottom"
                    class="w-4 h-4 text-btn-primary bg-dark-input border-dark-border focus:ring-btn-primary focus:ring-2" />
                  <span class="ml-2">Bottom of file (with sectional comment)</span>
                </label>
                <label class="flex items-center cursor-pointer">
                  <input type="radio" v-model="entryPlacement" value="preserve_order"
                    class="w-4 h-4 text-btn-primary bg-dark-input border-dark-border focus:ring-btn-primary focus:ring-2" />
                  <span class="ml-2">Preserve original order (experimental)</span>
                </label>
              </div>
            </div>

            <label class="flex items-center cursor-pointer ml-4">
              <input type="checkbox" v-model="useKeyList"
                class="w-4 h-4 text-btn-primary bg-dark-input border-dark-border rounded focus:ring-btn-primary focus:ring-2" />
              <span class="ml-2">
                Use key list to manually preserve specified B entries
              </span>
            </label>

            <div v-if="useKeyList" class="my-2 ml-8">
              <label class="block mb-2 font-medium">Entry Keys (one per line):</label>
              <textarea v-model="customKeys" rows="4" placeholder="event_name.0001&#10;event_name.0002"
                class="w-full px-3 py-2 bg-dark-input/80 border border-dark-border rounded-lg text-gray-200 focus:outline-none focus:ring-2 focus:ring-btn-primary/50 focus:border-btn-primary font-mono transition-all duration-200"></textarea>
            </div>
          </div>
        </div>

        <!-- File/Folder Selection Sections -->
        <div class="mb-6">
          <label class="block mb-4 text-xl font-semibold">File/Folder Selection</label>
          <div class="my-4">
            <label class="block mb-2 font-medium">
              File Set A (Base):
              <span v-if="mergeSetA.length > 0" class="text-sm text-gray-400 ml-2">
                ({{ mergeSetA.length }} {{ mergeSetA.length === 1 ? 'item' : 'items' }})
              </span>
            </label>
            <textarea :value="mergeSetA.join('\n')"
              @input="mergeSetA = $event.target.value.split('\n').filter(p => p.trim())" rows="3"
              placeholder="Select files or directories..."
              class="w-full px-3 py-2 bg-dark-input/80 border border-dark-border rounded-lg text-gray-200 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-btn-primary/50 focus:border-btn-primary font-mono text-sm transition-all duration-200"></textarea>
            <div class="flex gap-2 mt-2">
              <button @click="selectFiles('Select Multiple Files To Merge (A)', '*.txt', 'mergeSetA')"
                class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
                Select File(s)
              </button>
              <button @click="selectFolder('Select Folder To Merge (A)', 'mergeSetA')"
                class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
                Select Folder
              </button>
              <button @click="clearSet('mergeSetA')"
                class="px-4 py-2 bg-gray-600/80 hover:bg-gray-700/80 text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
                Clear
              </button>
            </div>
          </div>

          <div class="mb-4">
            <label class="block mb-2 font-medium">
              File Set B (Mod):
              <span v-if="mergeSetB.length > 0" class="text-sm text-gray-400 ml-2">
                ({{ mergeSetB.length }} {{ mergeSetB.length === 1 ? 'item' : 'items' }})
              </span>
            </label>
            <textarea :value="mergeSetB.join('\n')"
              @input="mergeSetB = $event.target.value.split('\n').filter(p => p.trim())" rows="3"
              placeholder="Select files or directories..."
              class="w-full px-3 py-2 bg-dark-input/80 border border-dark-border rounded-lg text-gray-200 placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-btn-primary/50 focus:border-btn-primary font-mono text-sm transition-all duration-200"></textarea>
            <div class="flex gap-2 mt-2">
              <button @click="selectFiles('Select Multiple Files To Merge (B)', '*.txt', 'mergeSetB')"
                class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
                Select File(s)
              </button>
              <button @click="selectFolder('Select Folder To Merge (B)', 'mergeSetB')"
                class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
                Select Folder
              </button>
              <button @click="clearSet('mergeSetB')"
                class="px-4 py-2 bg-gray-600/80 hover:bg-gray-700/80 text-white rounded-lg text-sm font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
                Clear
              </button>
            </div>
          </div>

          <div class="mb-4">
            <label class="block mb-2 font-medium">Output Directory:</label>
            <div class="flex gap-2">
              <input v-model="mergeOutputDir" type="text" placeholder="merger-output"
                class="flex-1 px-3 py-2 bg-dark-input/80 border border-dark-border rounded-lg text-gray-200 focus:outline-none focus:ring-2 focus:ring-btn-primary/50 focus:border-btn-primary transition-all duration-200" />
              <button @click="selectOutputDir"
                class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
                Browse
              </button>
            </div>
          </div>
        </div>

        <!-- Misc Options Sections -->
        <div class="mb-4">
          <label class="block mb-4 text-xl font-semibold">Misc Options</label>
          <div class="my-4">
            <label class="block mb-2 font-medium">Custom Comment Prefix:</label>
            <input v-model="commentPrefix" type="text" placeholder="# MOD:"
              class="w-full px-3 py-2 bg-dark-input/80 border border-dark-border rounded-lg text-gray-200 focus:outline-none focus:ring-2 focus:ring-btn-primary/50 focus:border-btn-primary transition-all duration-200" />
            <p class="text-sm text-gray-400 mt-2 leading-relaxed">Comments with above prefix will be preserved during
              merger.</p>
          </div>
        </div>

        <button @click="runMerge" :disabled="merging"
          class="w-full mt-6 px-4 py-3 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg font-medium transition-all duration-200 shadow-material hover:shadow-material-lg disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-material">
          {{ merging ? 'Merging...' : 'Run Merge' }}
        </button>
      </div>

      <!-- Results Panel -->
      <div v-if="!showComparisonTool && mergeResults.length > 0"
        class="bg-dark-panel/60 backdrop-blur-sm rounded-xl p-6 overflow-y-auto shadow-material border border-dark-border/50">
        <h2 class="text-xl font-semibold mb-5">Results</h2>
        <div v-for="(result, index) in mergeResults" :key="index"
          class="p-4 bg-dark-input/60 rounded-lg mb-3 border border-dark-border/30 shadow-material">
          <h3 class="text-base font-medium mb-2">{{ result.filePath }}</h3>
          <div class="flex gap-4 mb-2 text-sm text-gray-400">
            <span v-if="result.changed > 0">Changed: {{ result.changed }}</span>
            <span v-if="result.added > 0">Added: {{ result.added }}</span>
            <span v-if="result.removed > 0">Removed: {{ result.removed }}</span>
          </div>
          <div class="flex flex-col gap-2">
            <span>Merge Output: {{ getFileName(result.OutputPath) }}</span>
            <button @click="viewDiff(index, result.FileAPath, result.OutputPath)"
              class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
              View (A -> Output) Diff
            </button>
            <button @click="viewDiff(index, result.FileBPath, result.OutputPath)"
              class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
              View (B -> Output) Diff
            </button>
          </div>
        </div>
      </div>

      <!-- Diff Panel Overlay -->
      <DiffViewer :visible="diffFilePath && !showComparisonTool" :lines="diffLines"
        :loading="diffLines.length === 0 && diffFilePath" @close="closeDiff" />

      <!-- Comparison Tool Panel -->
      <ComparisonTool :visible="showComparisonTool" @view-report="viewCompToolReport" />
    </main>
  </div>
</template>

<script>
// Import App method bindings
import { SelectDirectory, SelectMultipleFiles, MergeMultipleFileSets, GetDiff } from '../wailsjs/go/main/App'
import DiffViewer from './components/DiffViewer.vue'
import ComparisonTool from './components/ComparisonTool.vue'

export default {
  name: 'App',
  components: {
    DiffViewer,
    ComparisonTool
  },
  data() {
    return {
      mergeSetA: [],
      mergeSetB: [],
      addAdditionalEntries: true,
      entryPlacement: 'bottom',
      useKeyList: false,
      customKeys: '',
      commentPrefix: '',
      mergeOutputDir: '',
      merging: false,
      mergeResults: [],
      diffLines: [],
      diffFilePath: null,
      showComparisonTool: false
    }
  },
  methods: {
    // Extract filename from path (handles both Windows and Unix paths)
    getFileName(path) {
      if (!path) return ''
      const parts = path.split(/[/\\]/)
      return parts[parts.length - 1] || path
    },
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
    async selectOutputDir() {
      try {
        const selected = await SelectDirectory('Select Output Directory')
        if (selected) {
          this.mergeOutputDir = selected
        }
      } catch (error) {
        alert('Error selecting directory: ' + error)
      }
    },
    async runMerge() {
      if (this.mergeSetA.length === 0 || this.mergeSetB.length === 0) {
        alert('Please select at least one file or directory for both sets')
        return
      }

      if (!this.mergeOutputDir) {
        alert('Please select an output directory')
        return
      }

      this.merging = true
      this.mergeResults = []

      try {
        // Build merge options
        const keys = this.useKeyList ? this.customKeys.split('\n').filter(k => k.trim() !== '') : []

        // Call backend to merge file sets
        const results = await MergeMultipleFileSets(
          this.mergeSetA,
          this.mergeSetB,
          this.mergeOutputDir,
          {
            AddAdditionalEntries: this.addAdditionalEntries,
            EntryPlacement: this.entryPlacement,
            KeyList: keys,
            CustomCommentPrefix: this.commentPrefix,
          }
        )

        this.mergeResults = results

        if (results.length === 0) {
          alert('No matching files found between the two sets. Make sure both sets contain files with matching names or paths.')
        }
      } catch (error) {
        alert('Error during merge: ' + error)
      } finally {
        this.merging = false
      }
    },
    async viewDiff(resultIndex, beforePath, afterPath) {
      const result = this.mergeResults[resultIndex]
      if (!result || result.Error) {
        alert('Cannot view diff: ' + (result?.Error || 'Unknown error'))
        return
      }

      try {
        const diffLines = await GetDiff(beforePath, afterPath)
        this.diffLines = diffLines || []
        this.diffFilePath = afterPath
      } catch (error) {
        alert('Error loading diff: ' + error)
        this.diffLines = []
      }
    },
    closeDiff() {
      this.diffLines = []
      this.diffFilePath = null
    },
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
