<template>
  <div v-if="visible" class="fixed left-0 right-0 bottom-0 top-20 bg-dark-bg/95 backdrop-blur-sm z-[60] flex flex-col">
    <div
      class="flex-1 flex flex-col m-4 bg-dark-panel/95 backdrop-blur-md rounded-xl shadow-material-lg border border-dark-border overflow-hidden">
      <div class="flex justify-between items-center px-6 py-4 border-b border-dark-border bg-dark-panel/50">
        <h2 class="text-xl font-semibold">Diff Viewer</h2>
        <button @click="$emit('close')"
          class="px-4 py-2 bg-btn-primary hover:bg-btn-primary-hover text-white rounded-lg font-medium transition-all duration-200 shadow-material hover:shadow-material-lg">
          Close
        </button>
      </div>
      <div class="flex-1 overflow-auto">
        <div v-if="loading" class="p-8 text-center text-gray-400">
          Loading diff...
        </div>
        <div v-else-if="lines.length > 0" class="font-mono text-sm leading-6"
          style="contain: layout style paint; will-change: scroll-position;">
          <div v-for="(line, index) in lines" :key="index" class="flex min-h-6 border-l-[3px]"
            :class="getLineClasses(line.type)" style="contain: layout style;">
            <!-- Line Numbers -->
            <div
              class="flex min-w-32 border-r border-dark-border/50 bg-dark-input/50 px-3 items-center justify-end gap-4 select-none flex-shrink-0">
              <span class="inline-block min-w-10 text-right tabular-nums text-slate-400"
                :class="{ 'text-gray-500': !line.oldLineNum }">
                {{ formatLineNum(line.oldLineNum) }}
              </span>
              <span class="inline-block min-w-10 text-right tabular-nums text-slate-400"
                :class="{ 'text-gray-500': !line.newLineNum }">
                {{ formatLineNum(line.newLineNum) }}
              </span>
            </div>
            <!-- Line Content -->
            <div class="flex-1 flex px-3 items-center overflow-x-auto whitespace-pre" style="contain: layout;">
              <span v-if="line.type !== 'header' && line.type !== 'other'"
                class="inline-block min-w-4 mr-2 font-semibold select-none" :class="getPrefixColor(line.type)">
                {{ getPrefix(line.type) }}
              </span>
              <span class="flex-1 break-all">{{ line.content }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'DiffViewer',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    lines: {
      type: Array,
      default: () => []
    },
    loading: {
      type: Boolean,
      default: false
    }
  },
  methods: {
    formatLineNum(num) {
      return num !== null && num !== undefined ? num : ''
    },
    getPrefix(type) {
      return type === 'add' ? '+' : type === 'remove' ? '-' : ' '
    },
    getPrefixColor(type) {
      return type === 'add'
        ? 'text-emerald-500/90'
        : type === 'remove'
          ? 'text-red-500/90'
          : ''
    },
    getLineClasses(type) {
      const typeClasses = {
        'add': 'bg-diff-add border-l-emerald-500/50',
        'remove': 'bg-diff-remove border-l-red-500/50',
        'header': 'bg-diff-header border-l-transparent',
        'context': 'bg-diff-context border-l-transparent'
      }
      return typeClasses[type] || 'border-l-transparent'
    }
  }
}
</script>
