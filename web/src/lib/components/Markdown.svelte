<script lang="ts">
	import DOMPurify from 'dompurify';
	import { marked } from 'marked';

	let { source }: { source: string } = $props();
	let rendered = $derived(DOMPurify.sanitize(marked.parse(source, { async: false })));
</script>

<!-- eslint-disable-next-line svelte/no-at-html-tags -- rendered is sanitized by DOMPurify -->
<div class="prose prose-sm max-w-none dark:prose-invert">{@html rendered}</div>
