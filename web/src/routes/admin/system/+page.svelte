<script lang="ts">
	import { goto } from '$app/navigation';
	import SystemWorkspace from './SystemWorkspace.svelte';
	import { resolve } from '$app/paths';
</script>

<svelte:head>
	<title>System | CloudJam</title>
	<meta property="og:title" content="System | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<SystemWorkspace close={() => goto(resolve('/admin/'), { keepFocus: true, noScroll: true })} />
