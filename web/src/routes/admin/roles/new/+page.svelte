<script lang="ts">
	import { goto } from '$app/navigation';
	import Button from '$lib/components/Button.svelte';
	import { ChevronLeftIcon } from '@lucide/svelte';
	import CreateRole from './components/CreateRole.svelte';
	import { resolve } from '$app/paths';
</script>

<svelte:head>
	<title>Create Role | CloudJam</title>
	<meta property="og:title" content="Create Role | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" onclick={() => goto(resolve('/admin/roles'))}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">Create Role</h1>
	</div>

	<CreateRole onCreated={() => goto(resolve('/admin/roles'))} />
</div>
