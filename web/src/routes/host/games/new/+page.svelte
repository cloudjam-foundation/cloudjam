<script lang="ts">
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import CreateGame from './CreateGame.svelte';
	import Button from '$lib/components/custom/buttons/Button.svelte';
	import { goto } from '$app/navigation';
</script>

<svelte:head>
	<title>Host Game | CloudJam</title>
	<meta property="og:title" content="Host Game | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" class="cursor-pointer" onclick={() => goto('/host/games')}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">Host Game</h1>
	</div>

	<CreateGame />
</div>
