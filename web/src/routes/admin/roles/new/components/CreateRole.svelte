<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import OptionalSelect from '$lib/components/OptionalSelect.svelte';
	import Card from '$lib/components/Card.svelte';
	import { getAssignableTenants } from '$lib/auth.svelte.js';
	import { CreateRequestSchema } from '$lib/sdk/v1/admin/role/role_pb';
	import { RoleSchema } from '$lib/sdk/v1/admin/role_pb';
	import { create } from '@bufbuild/protobuf';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	let configureState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	let id = $state(crypto.randomUUID());
	let name = $state('');
	let description = $state('');
	let tenant = $state('');

	let init = $derived(
		create(RoleSchema, {
			id: id,
			name: name,
			description: description,
			scope: tenant
		})
	);
</script>

<Card class="w-full items-start">
	<div>
		<h1 class="text-xl font-semibold text-primary/80">Create Role</h1>
		<p class="text-primary/60">
			Create a new role. To create roles in a different tenant, an administrator must first grant your role access to
			that tenant.
		</p>
	</div>
	<form
		class="flex w-full flex-col gap-4"
		onsubmit={() =>
			Submit(async () => {
				const resp = await Glue.role.create(create(CreateRequestSchema, { init: init }));
				await goto(resolve('/admin/roles/[id]', { id: resp.id }));
			}, configureState)}
	>
		<div class="grid gap-4 md:grid-cols-2">
			<Input
				bind:value={name}
				label="Name"
				placeholder="Name of the new role"
				validation={Glue.Validate(RoleSchema, init).violation.name}
			/>
			<Input
				bind:value={description}
				label="Description"
				placeholder="Description of the new role"
				validation={Glue.Validate(RoleSchema, init).violation.description}
			/>
			<div class="flex flex-col gap-1">
				<label class="text-sm">
					Tenant
					<OptionalSelect
						class="w-full"
						bind:value={tenant}
						placeholder="Tenant the role is placed in"
						suggestions={getAssignableTenants().map((tenant) => ({ id: tenant, title: tenant }))}
					/>
				</label>
			</div>
		</div>

		<Button
			type="submit"
			variant="primary"
			class="self-start"
			processing={configureState.loading}
			disabled={configureState.loading}
		>
			Create and Configure
		</Button>
		{#if configureState.error}
			<Alert
				title={configureState.forbidden ? 'Permission denied' : 'Failed to create role'}
				message={configureState.error}
				animation={true}
			/>
		{/if}
	</form>
</Card>
