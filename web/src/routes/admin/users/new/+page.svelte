<script lang="ts">
	import { goto } from '$app/navigation';
	import CreateUser from './components/CreateUser.svelte';
	import { resolve } from '$app/paths';
	import StandardPage from '$lib/components/ui/StandardPage.svelte';
</script>

<svelte:head>
	<title>Invite User | CloudJam</title>
	<meta property="og:title" content="Invite User | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<StandardPage title="Invite User" back={() => goto(resolve('/admin/users'))}>
	<CreateUser />
</StandardPage>
