<script lang="ts">
	import Card from '$lib/components/layout/Card.svelte';
	import GamepadIcon from '@lucide/svelte/icons/gamepad-2';
	import WandSparklesIcon from '@lucide/svelte/icons/wand-sparkles';
</script>

<svelte:head>
	<title>Host | CloudJam</title>
	<meta property="og:title" content="Host | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full flex-col items-center gap-4">
	<h1 class="my-8 text-4xl opacity-80">Host</h1>
	<div class="grid w-full gap-4 md:grid-cols-2">
		<a href="/host/games/">
			<Card class="cursor-pointer items-start gap-2 duration-300 hover:opacity-70">
				<GamepadIcon class="size-8" />
				<h1 class="text-2xl">Games</h1>
				<p>Schedule game events</p>
			</Card>
		</a>
		<a href="/host/design/">
			<Card class="cursor-pointer items-start gap-2 duration-300 hover:opacity-70">
				<WandSparklesIcon class="size-8" />
				<h1 class="text-2xl">Design</h1>
				<p>Design and upload your challenges</p>
			</Card>
		</a>
	</div>
</div>
