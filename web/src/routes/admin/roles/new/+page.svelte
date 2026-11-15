<script lang="ts">
	import { goto } from '$app/navigation';
	import CreateRole from './components/CreateRole.svelte';
	import { resolve } from '$app/paths';
	import StandardPage from '$lib/components/ui/StandardPage.svelte';
</script>

<svelte:head>
	<title>Create Role | CloudJam</title>
	<meta property="og:title" content="Create Role | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<StandardPage title="Create Role" back={() => goto(resolve('/admin/roles'))}>
	<CreateRole />
</StandardPage>
