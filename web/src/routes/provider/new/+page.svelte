<script lang="ts">
	import { goto } from '$app/navigation';
	import Button from '$lib/components/Button.svelte';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import CreateProvider from './CreateProvider.svelte';
	import { resolve } from '$app/paths';
</script>

<svelte:head>
	<title>Add Provider | CloudJam</title>
	<meta property="og:title" content="Add Provider | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" onclick={() => goto(resolve('/provider'))}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">Add Provider</h1>
	</div>

	<CreateProvider />
</div>
