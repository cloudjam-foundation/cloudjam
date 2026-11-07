<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import OptionalSelect from '$lib/components/OptionalSelect.svelte';
	import Card from '$lib/components/Card.svelte';
	import { getAssignableScopes } from '$lib/auth.svelte';
	import { ConfigureRoleRequestSchema } from '$lib/sdk/v1/admin/rbac/rbac_pb';
	import { CreateRequestSchema } from '$lib/sdk/v1/admin/role/role_pb';
	import { RoleSchema } from '$lib/sdk/v1/admin/role_pb';
	import { create } from '@bufbuild/protobuf';
	import PermissionEditor from '../PermissionEditor.svelte';

	let { oncreated }: { oncreated: () => void } = $props();

	let configureState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	let id = $state(crypto.randomUUID());
	let name = $state('');
	let description = $state('');
	let scope = $state('');
	let entries: { scope: string; patterns: string }[] = $state([{ scope: '', patterns: '' }]);

	let init = $derived(
		create(RoleSchema, {
			id: id,
			name: name,
			description: description,
			scope: scope,
			permissions: Object.fromEntries(entries.filter((e) => e.scope).map((e) => [e.scope, e.patterns]))
		})
	);
</script>

<Card class="w-full items-start">
	<div>
		<h1 class="text-xl font-semibold text-primary/80">Create Role</h1>
		<p class="text-primary/60">
			Creates a new role in the provided scope; permissions grant access by glob matching rpc procedure names against
			the patterns of every scope the requestor possesses.
		</p>
	</div>
	<form
		class="flex w-full flex-col gap-4"
		onsubmit={() =>
			Submit(async () => {
				await Glue.role.create(create(CreateRequestSchema, { init: init }));
				// create only persists the metadata, the permission map
				// is configured over the dedicated rbac endpoint.
				await Glue.rbac.configureRole(create(ConfigureRoleRequestSchema, { mod: init }));
				id = crypto.randomUUID();
				name = description = scope = '';
				entries = [{ scope: '', patterns: '' }];
				oncreated();
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
					Scope
					<OptionalSelect
						bind:value={scope}
						placeholder="Scope the role is placed in"
						suggestions={getAssignableScopes().map((scope) => ({ id: scope, title: scope }))}
					/>
				</label>
			</div>
		</div>
		<div class="flex flex-col gap-1">
			<span class="text-sm">Permissions</span>
			<PermissionEditor bind:entries />
		</div>
		<Button
			type="submit"
			variant="primary"
			class="self-start"
			processing={configureState.loading}
			disabled={configureState.loading || Boolean(Glue.Validate(RoleSchema, init).error)}
		>
			Create
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
