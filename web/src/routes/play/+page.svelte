<script lang="ts">
	import { goto } from '$app/navigation';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import Card from '$lib/components/Card.svelte';
	import { resolve } from '$app/paths';

	let code = $state('');
</script>

<svelte:head>
	<title>Play | CloudJam</title>
	<meta property="og:title" content="Play | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full flex-col gap-4">
	<div class="flex w-full justify-center">
		<Card class="mt-[10%] w-96 items-start">
			<div>
				<h1 class="text-2xl font-bold text-primary/80">Join a Game</h1>
				<p class="text-primary/60">Enter the game code your host handed you.</p>
			</div>
			<form
				class="flex w-full flex-col gap-4"
				onsubmit={(e) => {
					e.preventDefault();
					goto(resolve(`/play/[game_id]`, { game_id: code }));
				}}
			>
				<Input bind:value={code} placeholder="Game code" />
				<Button type="submit" variant="primary" disabled={!code.trim()}>Join</Button>
			</form>
		</Card>
	</div>
</div>
