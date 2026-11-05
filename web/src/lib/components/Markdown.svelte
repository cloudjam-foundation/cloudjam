<script lang="ts">
	import DOMPurify from 'dompurify';
	import { Marked } from 'marked';
	import { markedHighlight } from 'marked-highlight';
	import hljs from 'highlight.js';
	import light from 'highlight.js/styles/vs.css?inline';
	import dark from 'highlight.js/styles/vs-dark.css?inline';

	const themeCss = `<style> :root:not(.dark) { ${light} } :root.dark { ${dark} } </style>`;

	const marked = new Marked(
		markedHighlight({
			emptyLangClass: 'hljs',
			langPrefix: 'hljs language-',
			highlight(code, lang) {
				const language = hljs.getLanguage(lang) ? lang : 'plaintext';
				return hljs.highlight(code, { language }).value;
			}
		})
	);

	let { source }: { source: string } = $props();
	let rendered = $derived(DOMPurify.sanitize(marked.parse(source, { async: false })));
</script>

<svelte:head>
	{@html themeCss}
</svelte:head>

<!-- eslint-disable-next-line svelte/no-at-html-tags -- rendered is sanitized by DOMPurify -->
<article class="prose max-w-none">{@html rendered}</article>
