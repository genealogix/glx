<script setup>
const props = defineProps({
  content: {
    type: String,
    required: true
  },
  title: {
    type: String,
    default: ''
  }
})
</script>

<template>
  <div class="yaml-file">
    <div v-if="title" class="yaml-file-header">
      <h4>{{ title }}</h4>
    </div>
    <div class="yaml-file-content">
      <div class="language-yaml vp-adaptive-theme">
        <button title="Copy Code" class="copy"></button>
        <span class="lang">yaml</span>
        <pre class="shiki shiki-themes github-light github-dark vp-code"><code>{{ content }}</code></pre>
      </div>
    </div>
  </div>
</template>

<style scoped>
.yaml-file {
  margin: 1rem 0;
}

.yaml-file-header h4 {
  margin: 0 0 0.5rem 0;
  font-size: 1.1rem;
  font-weight: 600;
}

.yaml-file-content {
  position: relative;
}

/* Match VitePress code block styling */
.yaml-file-content .language-yaml {
  position: relative;
  margin: 16px 0;
  background-color: var(--vp-code-block-bg);
  overflow-x: auto;
  transition: background-color 0.5s;
  border-radius: 8px;
}

.yaml-file-content pre {
  margin: 0;
  padding: 20px 24px;
  overflow-x: auto;
  line-height: 1.5;
}

.yaml-file-content code {
  display: block;
  width: fit-content;
  min-width: 100%;
  font-family: var(--vp-font-family-mono);
  font-size: 14px;
  color: var(--vp-code-block-color);
  transition: color 0.5s;
}

.yaml-file-content .lang {
  position: absolute;
  top: 8px;
  right: 12px;
  z-index: 2;
  font-size: 12px;
  font-weight: 500;
  color: var(--vp-c-text-3);
  transition: color 0.5s;
}

.yaml-file-content .copy {
  position: absolute;
  top: 8px;
  right: 50px;
  z-index: 3;
  border: 1px solid var(--vp-code-copy-code-border-color);
  border-radius: 4px;
  width: 40px;
  height: 40px;
  background-color: var(--vp-code-copy-code-bg);
  opacity: 0;
  cursor: pointer;
  background-image: var(--vp-icon-copy);
  background-position: 50%;
  background-size: 20px;
  background-repeat: no-repeat;
  transition: border-color 0.25s, background-color 0.25s, opacity 0.25s;
}

.yaml-file-content:hover .copy {
  opacity: 1;
}

.yaml-file-content .copy:hover {
  border-color: var(--vp-code-copy-code-hover-border-color);
  background-color: var(--vp-code-copy-code-hover-bg);
}

.yaml-file-content .copy.copied {
  border-radius: 0 4px 4px 0;
  background-color: var(--vp-code-copy-code-hover-bg);
  background-image: var(--vp-icon-copied);
}
</style>

