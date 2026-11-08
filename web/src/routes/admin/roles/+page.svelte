<script lang="ts">
	import { goto } from '$app/navigation';
	import { Glue, Submit, type SubmitState } from '$lib';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import LinkButton from '$lib/components/LinkButton.svelte';
	import Input from '$lib/components/Input.svelte';
	import Card from '$lib/components/Card.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import { ListRequestSchema } from '$lib/sdk/v1/admin/role/role_pb';
	import type { Role } from '$lib/sdk/v1/admin/role_pb';
	import { create } from '@bufbuild/protobuf';
	import { ChevronLeftIcon, PlusIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';

	let id = $state('');

	let rolesState: SubmitState = $state({ error: '', loading: true, forbidden: false });
</script>

<svelte:head>
	<title>Roles | CloudJam</title>
	<meta property="og:title" content="Roles | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex h-11/12 w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" onclick={() => goto(resolve('/admin'))}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">Roles</h1>
		<LinkButton class="ml-auto" href="/admin/roles/new/">
			<PlusIcon /> Create Role
		</LinkButton>
	</div>

	{#if rolesState.forbidden}
		<div class="flex h-full w-full items-center justify-center">
			<Card>
				<div class="flex flex-col items-start gap-1">
					<h1 class="text-2xl font-bold text-primary/80">Find Role</h1>
					<p class="text-primary/60">You are not allowed to list roles, find the one you need by its id</p>
				</div>
				<form class="flex w-full flex-row items-center gap-2" onsubmit={() => goto(resolve(`/admin/roles/${id}`))}>
					<Input class="w-full" bind:value={id} placeholder="Role ID" />
					<Button type="submit" variant="primary" disabled={!id.trim()}>Open</Button>
				</form>
			</Card>
		</div>
	{:else}
		<div class="h-full rounded-sm p-2">
			<DataTable
				class="max-h-full"
				limit={100}
				load={async (startAfter: Role | undefined) => {
					const roles: Role[] = [];
					await Submit(async () => {
						const resp = await Glue.role.list(create(ListRequestSchema, { limit: 100, startAfter: startAfter?.id }));
						roles.push(...resp.roles);
					}, rolesState);
					return roles;
				}}
				onRowClick={(role) => {
					goto(resolve(`/admin/roles/${role.id}`));
				}}
				loadState={rolesState}
				columns={[
					{ title: 'Name', width: '25%', render: renderName },
					{ title: 'Scope', width: '20%', render: renderScope },
					{ title: 'Permissions', width: '15%', render: renderPermissions },
					{ title: 'Id', width: '30%', render: renderId },
					{ title: '', width: '90px', render: renderBuiltin }
				]}
			/>
			{#snippet renderName(role: Role)}
				{role.name}
			{/snippet}
			{#snippet renderScope(role: Role)}
				{role.scope}
			{/snippet}
			{#snippet renderPermissions(role: Role)}
				<span class="text-primary/60">{Object.keys(role.permissions).length} scope(s)</span>
			{/snippet}
			{#snippet renderId(role: Role)}
				<span class="font-mono text-xs">{role.id}</span>
			{/snippet}
			{#snippet renderBuiltin(role: Role)}
				{#if role.builtin}
					<StaticBadge variant="jam" text="builtin" />
				{/if}
			{/snippet}
		</div>
	{/if}
</div>
