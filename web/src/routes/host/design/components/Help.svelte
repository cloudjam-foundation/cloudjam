<script lang="ts">
	import Button from '$lib/components/Button.svelte';
	import { CircleQuestionMarkIcon } from '@lucide/svelte';
	import Popup from '$lib/components/Popup.svelte';
	import Markdown from '$lib/components/Markdown.svelte';
	import HelpMarkdown from '../Help.md?raw';

	let { class: className }: { class: string } = $props();

	let open = $state(false);
</script>

<Button variant="primary" class={className} onclick={() => (open = true)}>
	<CircleQuestionMarkIcon />
	Help
</Button>

<Popup bind:open title="" description="" body={help} />

{#snippet help()}
	<Markdown source={HelpMarkdown} />
{/snippet}
