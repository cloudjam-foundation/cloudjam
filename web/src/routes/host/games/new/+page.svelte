<script lang="ts">
	import CreateGame from './components/CreateGame.svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import StandardPage from '$lib/components/ui/StandardPage.svelte';
</script>

<svelte:head>
	<title>Host Game | CloudJam</title>
	<meta property="og:title" content="Host Game | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<StandardPage title="Host Game" back={() => goto(resolve('/host/games'))}>
	<CreateGame />
</StandardPage>
