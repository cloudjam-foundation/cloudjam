<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import { ListRequestSchema } from '$lib/sdk/v1/admin/user/user_pb';
	import type { User } from '$lib/sdk/v1/admin/user_pb';
	import { create } from '@bufbuild/protobuf';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import { toSvg } from 'jdenticon';
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import Input from '$lib/components/Input.svelte';
	import { goto } from '$app/navigation';
	import LinkButton from '$lib/components/LinkButton.svelte';
	import DataTable from '$lib/components/DataTable.svelte';

	let id = $state('');

	let usersState: SubmitState = $state({ error: '', loading: true, forbidden: false });
</script>

<svelte:head>
	<title>Users | CloudJam</title>
	<meta property="og:title" content="Users | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex h-11/12 w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" onclick={() => goto('/admin')}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">Users</h1>
		<LinkButton class="ml-auto" href="/admin/users/new/">
			<PlusIcon /> Invite User
		</LinkButton>
	</div>

	{#if usersState.forbidden}
		<div class="flex h-full w-full items-center justify-center">
			<Card>
				<div class="flex flex-col items-start gap-1">
					<h1 class="text-2xl font-bold text-primary/80">Find User</h1>
					<p class="text-primary/60">You are not allowed to list users, find the one you need by its id</p>
				</div>
				<form class="flex w-full flex-row items-center gap-2" onsubmit={() => goto(`/admin/users/${id}`)}>
					<Input class="w-full" bind:value={id} placeholder="User ID" />
					<Button type="submit" variant="primary" disabled={!id.trim()}>Open</Button>
				</form>
			</Card>
		</div>
	{:else}
		<div class="h-full rounded-sm p-2">
			<DataTable
				class="max-h-full"
				limit={100}
				load={async (startAfter: User | undefined) => {
					const users: User[] = [];
					await Submit(async () => {
						const resp = await Glue.user.list(create(ListRequestSchema, { limit: 100, startAfter: startAfter?.id }));
						users.push(...resp.users);
					}, usersState);
					return users;
				}}
				onRowClick={(user) => {
					goto(`/admin/users/${user.id}`);
				}}
				loadState={usersState}
				columns={[
					{ title: '#', width: '4rem', render: renderIcon },
					{ title: 'Username', width: '25%', render: renderUsername },
					{ title: 'Email', width: '25%', render: renderEmail },
					{ title: 'Organization', width: '25%', render: renderOrganization },
					{ title: 'Role', width: '25%', render: renderRole }
				]}
			/>
			{#snippet renderIcon(user: User)}
				<img
					alt="user profile"
					src={`data:image/svg+xml;base64,${btoa(toSvg(user.pubId, 30))}`}
					height="4rem"
					class="rounded-md bg-primary/5"
				/>
			{/snippet}
			{#snippet renderUsername(user: User)}
				{user.username}
			{/snippet}
			{#snippet renderEmail(user: User)}
				{user.email}
			{/snippet}
			{#snippet renderOrganization(user: User)}
				{user.organization}
			{/snippet}
			{#snippet renderRole(user: User)}
				{user.role}
			{/snippet}
		</div>
	{/if}
</div>
