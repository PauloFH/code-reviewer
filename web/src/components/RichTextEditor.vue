<template>
  <div class="editor-wrap">
    <div v-if="editor" class="toolbar">
      <button type="button" :class="{ active: editor.isActive('bold') }" @click="editor.chain().focus().toggleBold().run()">B</button>
      <button type="button" :class="{ active: editor.isActive('italic') }" @click="editor.chain().focus().toggleItalic().run()">I</button>
      <span class="sep" />
      <button type="button" :class="{ active: editor.isActive('heading', { level: 2 }) }" @click="editor.chain().focus().toggleHeading({ level: 2 }).run()">H2</button>
      <button type="button" :class="{ active: editor.isActive('heading', { level: 3 }) }" @click="editor.chain().focus().toggleHeading({ level: 3 }).run()">H3</button>
      <span class="sep" />
      <button type="button" :class="{ active: editor.isActive('bulletList') }" @click="editor.chain().focus().toggleBulletList().run()">&#8226; List</button>
      <button type="button" :class="{ active: editor.isActive('orderedList') }" @click="editor.chain().focus().toggleOrderedList().run()">1. List</button>
      <span class="sep" />
      <button type="button" :class="{ active: editor.isActive('codeBlock') }" @click="editor.chain().focus().toggleCodeBlock().run()">Code</button>
    </div>
    <EditorContent :editor="editor" class="editor-content" />
  </div>
</template>

<script setup lang="ts">
import { watch, onBeforeUnmount } from 'vue'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'

const props = defineProps<{
  modelValue: string
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editor = useEditor({
  content: props.modelValue,
  extensions: [
    StarterKit,
    Placeholder.configure({ placeholder: props.placeholder || 'Escreva aqui...' }),
  ],
  onUpdate: ({ editor: e }) => {
    emit('update:modelValue', e.getHTML())
  },
})

watch(() => props.modelValue, (val) => {
  if (editor.value && editor.value.getHTML() !== val) {
    editor.value.commands.setContent(val, { emitUpdate: false })
  }
})

onBeforeUnmount(() => {
  editor.value?.destroy()
})
</script>

<style scoped>
.editor-wrap {
  border: 1px solid #475569;
  border-radius: 8px;
  overflow: hidden;
  background: #0f172a;
}
.toolbar {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 6px 8px;
  background: #1e293b;
  border-bottom: 1px solid #334155;
  flex-wrap: wrap;
}
.toolbar button {
  padding: 4px 8px;
  border: none;
  background: transparent;
  color: #94a3b8;
  border-radius: 4px;
  cursor: pointer;
  font-size: .75rem;
  font-weight: 600;
}
.toolbar button:hover { background: #334155; color: #e2e8f0; }
.toolbar button.active { background: #6366f1; color: white; }
.sep { width: 1px; height: 18px; background: #334155; margin: 0 4px; }
.editor-content {
  padding: 12px;
  min-height: 150px;
  color: #e2e8f0;
  font-size: .875rem;
  line-height: 1.6;
}
.editor-content :deep(.tiptap) {
  outline: none;
  min-height: 120px;
}
.editor-content :deep(.tiptap p.is-editor-empty:first-child::before) {
  content: attr(data-placeholder);
  color: #475569;
  pointer-events: none;
  float: left;
  height: 0;
}
.editor-content :deep(h2) { font-size: 1.2rem; font-weight: 700; margin: .5rem 0; color: #f8fafc; }
.editor-content :deep(h3) { font-size: 1rem; font-weight: 600; margin: .4rem 0; color: #f1f5f9; }
.editor-content :deep(ul), .editor-content :deep(ol) { padding-left: 1.5rem; margin: .4rem 0; }
.editor-content :deep(li) { margin: .2rem 0; }
.editor-content :deep(code) { background: #1e293b; padding: 2px 4px; border-radius: 3px; font-size: .8rem; }
.editor-content :deep(pre) { background: #1e293b; padding: 12px; border-radius: 6px; overflow-x: auto; margin: .5rem 0; }
.editor-content :deep(pre code) { background: none; padding: 0; }
</style>
