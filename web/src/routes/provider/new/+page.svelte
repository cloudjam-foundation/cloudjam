<script lang="ts">
	import { goto } from '$app/navigation';
	import CreateProvider from './components/CreateProvider.svelte';
	import { resolve } from '$app/paths';
	import StandardPage from '$lib/components/ui/StandardPage.svelte';
</script>

<svelte:head>
	<title>Add Provider | CloudJam</title>
	<meta property="og:title" content="Add Provider | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<StandardPage title="Add Provider" back={() => goto(resolve('/provider'))}>
	<CreateProvider />
</StandardPage>
