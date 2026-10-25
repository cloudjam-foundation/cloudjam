<!-- Spinner that activates only after the specified delay to avoid bubble spinners if it loads fast -->
<script lang="ts">
	import { onMount } from 'svelte';
	import Spinner from '../shad/spinner/spinner.svelte';
	import { fade } from 'svelte/transition';

	let { delay, class: className }: { delay: number; class?: string } = $props();

	let spin = $state(false);

	onMount(() => {
		setTimeout(() => (spin = true), delay);
	});
</script>

{#if spin}
	<div transition:fade class={className}>
		<Spinner />
	</div>
{/if}
