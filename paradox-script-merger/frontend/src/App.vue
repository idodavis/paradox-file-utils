<template>
  <div class="flex flex-col h-screen bg-dark-bg text-gray-200">
    <header class="bg-dark-panel px-8 py-4 border-b border-dark-border flex justify-between items-center">
      <h1 class="text-3xl font-semibold">Paradox Script Merger</h1>
      <nav class="flex gap-2">
        <button @click="showPatchComparison = false" :class="[
          'px-4 py-2 rounded border transition-colors',
          !showPatchComparison
            ? 'bg-btn-primary border-btn-primary text-white'
            : 'bg-transparent border-dark-border text-gray-400 hover:border-gray-500'
        ]">
          Merge Files
        </button>
        <button @click="showPatchComparison = true" :class="[
          'px-4 py-2 rounded border transition-colors',
          showPatchComparison
            ? 'bg-btn-primary border-btn-primary text-white'
            : 'bg-transparent border-dark-border text-gray-400 hover:border-gray-500'
        ]">
          Patch Comparison
        </button>
      </nav>
    </header>

    <main class="flex-1 grid grid-cols-2 gap-4 p-4 overflow-auto bg-dark-input">
      <!-- Configuration Panel -->
      <div v-if="!showPatchComparison" class="bg-dark-panel rounded-lg p-6 overflow-y-auto">
        <!-- Merge Options -->
        <div class="mb-6">
          <div class="mb-4">
            <h3 class="text-lg mb-2">Merge Options:</h3>
            <p class="text-md text-gray-400 mb-4">File Set A (Base) entries take precedence in conflicts. To use B as
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
                class="w-full px-3 py-2 bg-dark-input border border-dark-border rounded text-gray-200 focus:outline-none focus:border-primary font-mono"></textarea>
            </div>
          </div>
        </div>

        <!-- File/Folder Selection Sections -->
        <div class="mb-6">
          <label class="block mb-2 text-3xl">File/Folder Selection:</label>
          <div class="my-4">
            <label class="block mb-2 font-medium">
              File Set A (Base):
              <span v-if="fileSetA.length > 0" class="text-sm text-gray-400 ml-2">
                ({{ fileSetA.length }} {{ fileSetA.length === 1 ? 'item' : 'items' }})
              </span>
            </label>
            <textarea :value="fileSetA.join('\n')"
              @input="fileSetA = $event.target.value.split('\n').filter(p => p.trim())" rows="3"
              placeholder="Select files or directories..."
              class="w-full px-3 py-2 bg-dark-input border border-dark-border rounded text-gray-200 placeholder:text-gray-400 focus:outline-none focus:border-primary font-mono text-sm"></textarea>
            <div class="flex gap-2 mt-2">
              <button @click="addFilesToSetA"
                class="px-3 py-1.5 bg-btn-primary hover:bg-btn-primary-hover text-white rounded text-sm font-medium transition-colors">
                Browse File/Folders
              </button>
              <button @click="clearSetA"
                class="px-3 py-1.5 bg-gray-600 hover:bg-gray-700 text-white rounded text-sm font-medium transition-colors">
                Clear
              </button>
            </div>
          </div>

          <div class="mb-4">
            <label class="block mb-2 font-medium">
              File Set B (Mod):
              <span v-if="fileSetB.length > 0" class="text-sm text-gray-400 ml-2">
                ({{ fileSetB.length }} {{ fileSetB.length === 1 ? 'item' : 'items' }})
              </span>
            </label>
            <textarea :value="fileSetB.join('\n')"
              @input="fileSetB = $event.target.value.split('\n').filter(p => p.trim())" rows="3"
              placeholder="Select files or directories..."
              class="w-full px-3 py-2 bg-dark-input border border-dark-border rounded text-gray-200 placeholder:text-gray-400 focus:outline-none focus:border-primary font-mono text-sm"></textarea>
            <div class="flex gap-2 mt-2">
              <button @click="addFilesToSetB"
                class="px-3 py-1.5 bg-btn-primary hover:bg-btn-primary-hover text-white rounded text-sm font-medium transition-colors">
                Browse File/Folders
              </button>
              <button @click="clearSetB"
                class="px-3 py-1.5 bg-gray-600 hover:bg-gray-700 text-white rounded text-sm font-medium transition-colors">
                Clear
              </button>
            </div>
          </div>

          <div class="mb-4">
            <label class="block mb-2 font-medium">Output Directory:</label>
            <div class="flex gap-2">
              <input v-model="outputDir" type="text" placeholder="merger-output"
                class="flex-1 px-3 py-2 bg-dark-input border border-dark-border rounded text-gray-200 focus:outline-none focus:border-primary" />
              <button @click="selectOutputDir"
                class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded font-medium transition-colors">
                Browse
              </button>
            </div>
          </div>
        </div>

        <!-- Misc Options Sections -->
        <div class="mb-4">
          <label class="block mb-2 text-3xl">Misc Options:</label>
          <div class="my-4">
            <label class="block mb-2 font-medium">Custom Comment Prefix:</label>
            <input v-model="commentPrefix" type="text" placeholder="# MOD:"
              class="w-full px-3 py-2 bg-dark-input border border-dark-border rounded text-gray-200 focus:outline-none focus:border-primary" />
            <p class="text-md text-gray-400 mt-2"> Comments with above prefix will be preserved during merger.</p>
          </div>
        </div>

        <button @click="runMerge" :disabled="merging"
          class="w-full mt-4 px-4 py-3 bg-btn-primary hover:bg-btn-primary-hover text-white rounded font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
          {{ merging ? 'Merging...' : 'Run Merge' }}
        </button>
      </div>

      <!-- Results Panel -->
      <div v-if="!showPatchComparison && mergeResults.length > 0" class="bg-dark-panel rounded-lg p-6 overflow-y-auto">
        <h2 class="text-xl font-semibold mb-4">Results</h2>
        <div v-for="(result, index) in mergeResults" :key="index" class="p-4 bg-dark-input rounded mb-2">
          <h3 class="text-base font-medium mb-2">{{ result.filePath }}</h3>
          <div class="flex gap-4 mb-2 text-sm text-gray-400">
            <span v-if="result.changed > 0">Changed: {{ result.changed }}</span>
            <span v-if="result.added > 0">Added: {{ result.added }}</span>
            <span v-if="result.removed > 0">Removed: {{ result.removed }}</span>
          </div>
          <button @click="viewDiff(index)"
            class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded font-medium transition-colors">
            View Diff
          </button>
        </div>
      </div>

      <!-- Diff Panel -->
      <div v-if="currentDiff && !showPatchComparison" class="col-span-2 bg-dark-panel rounded-lg p-6">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-semibold">Diff Viewer</h2>
          <button @click="closeDiff"
            class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded font-medium transition-colors">
            Close
          </button>
        </div>
        <pre
          class="bg-dark-input p-4 rounded overflow-x-auto font-mono text-sm leading-relaxed whitespace-pre-wrap break-words">{{ currentDiff }}</pre>
      </div>

      <!-- Patch Comparison Panel -->
      <div v-if="showPatchComparison" class="col-span-2 bg-dark-panel rounded-lg p-6">
        <h2 class="text-xl font-semibold mb-4">Patch Comparison</h2>
        <div class="mb-4">
          <label class="block mb-2 font-medium">Current Game Directory:</label>
          <div class="flex gap-2">
            <input v-model="patchDirA" type="text" placeholder="Select current game directory..."
              class="flex-1 px-3 py-2 bg-dark-input border border-dark-border rounded text-gray-200 focus:outline-none focus:border-primary" />
            <button @click="selectPatchDirA"
              class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded font-medium transition-colors">
              Browse
            </button>
          </div>
        </div>
        <div class="mb-4">
          <label class="block mb-2 font-medium">Previous Patch Directory:</label>
          <div class="flex gap-2">
            <input v-model="patchDirB" type="text" placeholder="Select previous patch directory..."
              class="flex-1 px-3 py-2 bg-dark-input border border-dark-border rounded text-gray-200 focus:outline-none focus:border-primary" />
            <button @click="selectPatchDirB"
              class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded font-medium transition-colors">
              Browse
            </button>
          </div>
        </div>
        <button @click="runPatchComparison" :disabled="comparingPatches"
          class="w-full px-4 py-3 bg-btn-primary hover:bg-btn-primary-hover text-white rounded font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
          {{ comparingPatches ? 'Comparing...' : 'Compare Patches' }}
        </button>
        <div v-if="patchComparison" class="mt-4 p-4 bg-dark-input rounded">
          <h3 class="text-lg font-semibold mb-2">Results</h3>
          <p class="mb-1">Changed: {{ patchComparison.ChangedFiles?.length || 0 }}</p>
          <p class="mb-1">Added: {{ patchComparison.AddedFiles?.length || 0 }}</p>
          <p class="mb-4">Removed: {{ patchComparison.RemovedFiles?.length || 0 }}</p>
          <button @click="viewPatchReport"
            class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded font-medium transition-colors">
            View Full Report
          </button>
        </div>
      </div>
    </main>
  </div>
</template>

<script>
// Import App method bindings
import { SelectDirectory, SelectMultipleFiles, MergeMultipleFileSets, GetDiffForFile, ComparePatches, GeneratePatchReport } from '../wailsjs/go/main/App'

export default {
  name: 'App',
  data() {
    return {
      fileSetA: [],
      fileSetB: [],
      addAdditionalEntries: true,
      entryPlacement: 'bottom',
      useKeyList: false,
      customKeys: '',
      commentPrefix: '',
      outputDir: '',
      merging: false,
      mergeResults: [],
      currentDiff: null,
      diffFilePath: null,
      showPatchComparison: false,
      patchDirA: '',
      patchDirB: '',
      comparingPatches: false,
      patchComparison: null
    }
  },
  methods: {
    async addFilesToSetA() {
      try {
        const selected = await SelectMultipleFiles('Select Files for Set A (Base)', '*.txt')
        if (selected && selected.length > 0) {
          // Add new files, avoiding duplicates
          const existing = new Set(this.fileSetA)
          for (const file of selected) {
            if (!existing.has(file)) {
              this.fileSetA.push(file)
            }
          }
        }
      } catch (error) {
        alert('Error selecting files: ' + error)
      }
    },
    async addDirectoryToSetA() {
      try {
        const selected = await SelectDirectory('Select Directory for Set A (Base)')
        if (selected) {
          // Add directory if not already in list
          if (!this.fileSetA.includes(selected)) {
            this.fileSetA.push(selected)
          }
        }
      } catch (error) {
        alert('Error selecting directory: ' + error)
      }
    },
    clearSetA() {
      this.fileSetA = []
    },
    async addFilesToSetB() {
      try {
        const selected = await SelectMultipleFiles('Select Files for Set B (Mod)', '*.txt')
        if (selected && selected.length > 0) {
          // Add new files, avoiding duplicates
          const existing = new Set(this.fileSetB)
          for (const file of selected) {
            if (!existing.has(file)) {
              this.fileSetB.push(file)
            }
          }
        }
      } catch (error) {
        alert('Error selecting files: ' + error)
      }
    },
    async addDirectoryToSetB() {
      try {
        const selected = await SelectDirectory('Select Directory for Set B (Mod)')
        if (selected) {
          // Add directory if not already in list
          if (!this.fileSetB.includes(selected)) {
            this.fileSetB.push(selected)
          }
        }
      } catch (error) {
        alert('Error selecting directory: ' + error)
      }
    },
    clearSetB() {
      this.fileSetB = []
    },
    async selectOutputDir() {
      try {
        const selected = await SelectDirectory('Select Output Directory')
        if (selected) {
          this.outputDir = selected
        }
      } catch (error) {
        alert('Error selecting directory: ' + error)
      }
    },
    async runMerge() {
      if (!this.fileSetA || this.fileSetA.length === 0 || !this.fileSetB || this.fileSetB.length === 0) {
        alert('Please select at least one file or directory for both sets')
        return
      }

      if (!this.outputDir) {
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
          this.fileSetA,
          this.fileSetB,
          this.outputDir,
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
    async viewDiff(index) {
      const result = this.mergeResults[index]
      if (!result || result.Error) {
        alert('Cannot view diff: ' + (result?.Error || 'Unknown error'))
        return
      }

      if (!result.FileAPath || !result.FileBPath) {
        alert('File paths not available for diff')
        return
      }

      try {
        this.currentDiff = 'Loading diff...'
        const keys = this.useKeyList ? this.customKeys.split('\n').filter(k => k.trim() !== '') : []

        const diffContent = await GetDiffForFile(
          result.FileAPath,
          result.FileBPath,
          {
            AddAdditionalEntries: this.addAdditionalEntries,
            EntryPlacement: this.entryPlacement,
            KeyList: keys,
            CustomCommentPrefix: this.commentPrefix,
          }
        )

        this.currentDiff = diffContent
        this.diffFilePath = result.FilePath
      } catch (error) {
        alert('Error loading diff: ' + error)
        this.currentDiff = null
      }
    },
    closeDiff() {
      this.currentDiff = null
      this.diffFilePath = null
    },
    async selectPatchDirA() {
      try {
        const selected = await SelectDirectory('Select Current Game Directory')
        if (selected) {
          this.patchDirA = selected
        }
      } catch (error) {
        alert('Error selecting directory: ' + error)
      }
    },
    async selectPatchDirB() {
      try {
        const selected = await SelectDirectory('Select Previous Patch Directory')
        if (selected) {
          this.patchDirB = selected
        }
      } catch (error) {
        alert('Error selecting directory: ' + error)
      }
    },
    async runPatchComparison() {
      if (!this.patchDirA || !this.patchDirB) {
        alert('Please select both directories')
        return
      }

      this.comparingPatches = true
      this.patchComparison = null

      try {
        const comparison = await ComparePatches(this.patchDirA, this.patchDirB, ['.txt'])
        this.patchComparison = comparison
      } catch (error) {
        alert('Error comparing patches: ' + error)
      } finally {
        this.comparingPatches = false
      }
    },
    async viewPatchReport() {
      if (!this.patchComparison) {
        return
      }

      try {
        const report = await GeneratePatchReport(this.patchComparison)
        this.currentDiff = report
        this.diffFilePath = 'Patch Comparison Report'
      } catch (error) {
        alert('Error generating report: ' + error)
      }
    }
  }
}
</script>
