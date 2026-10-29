<script lang="ts">
	import UsersIcon from '@lucide/svelte/icons/users';
	import ShieldCheckIcon from '@lucide/svelte/icons/shield-check';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import Card from '$lib/components/layout/Card.svelte';
</script>

<svelte:head>
	<title>Admin | CloudJam</title>
	<meta property="og:title" content="Admin | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full flex-col items-center gap-4">
	<h1 class="my-8 text-4xl opacity-80">Host</h1>
	<div class="grid w-full gap-4 md:grid-cols-3">
		<a href="/admin/users/">
			<Card class="cursor-pointer items-start gap-2 duration-300 hover:opacity-70">
				<UsersIcon class="size-8" />
				<h1 class="text-2xl">Users</h1>
				<p>Invite, manage and scope the users of your hierarchy</p>
			</Card>
		</a>
		<a href="/admin/roles/">
			<Card class="cursor-pointer items-start gap-2 duration-300 hover:opacity-70">
				<ShieldCheckIcon class="size-8" />
				<h1 class="text-2xl">Roles</h1>
				<p>Create roles and configure their rbac permission policies</p>
			</Card>
		</a>
		<a href="/admin/system/">
			<Card class="cursor-pointer items-start gap-2 duration-300 hover:opacity-70">
				<ActivityIcon class="size-8" />
				<h1 class="text-2xl">System</h1>
				<p>Observe system logs, metrics and cost analysis</p>
			</Card>
		</a>
	</div>
</div>
