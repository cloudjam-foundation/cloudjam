<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import OptionalSelect from '$lib/components/OptionalSelect.svelte';
	import Card from '$lib/components/Card.svelte';
	import Separator from '$lib/components/Separator.svelte';
	import { getAssignableScopes } from '$lib/auth.svelte.js';
	import { AttachScopeRequestSchema, ConfigureRoleRequestSchema, Resource } from '$lib/sdk/v1/admin/rbac/rbac_pb';
	import { DeleteRequestSchema, UpdateRequestSchema } from '$lib/sdk/v1/admin/role/role_pb';
	import { Feature, RoleSchema, type Role } from '$lib/sdk/v1/admin/role_pb';
	import { create } from '@bufbuild/protobuf';
	import PermissionEditor from '../../components/PermissionEditor.svelte';
	import FeatureEditor from './FeatureEditor.svelte';

	let { role, refresh, deleted }: { role: Role; refresh: () => void; deleted: () => void } = $props();

	// the panel is remounted (keyed) per role, capturing the initial value is intended.
	// svelte-ignore state_referenced_locally
	let name = $state(role.name);
	// svelte-ignore state_referenced_locally
	let description = $state(role.description);
	// svelte-ignore state_referenced_locally
	let features: Record<Feature, boolean> = $state(
		Object.fromEntries(role.features.map((feature) => [feature, true])) as Record<Feature, boolean>
	);
	let attachScope = $state('');
	let confirmDelete = $state(false);
	// svelte-ignore state_referenced_locally
	let entries: { scope: string; patterns: string }[] = $state(
		Object.entries(role.permissions).map(([scope, patterns]) => ({ scope, patterns }))
	);

	let updateState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let configureState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let attachState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let removeState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	let mod = $derived(
		create(RoleSchema, {
			...role,
			name: name,
			description: description,
			features: Object.entries(features)
				// eslint-disable-next-line @typescript-eslint/no-unused-vars
				.filter(([_, active]) => active)
				// eslint-disable-next-line @typescript-eslint/no-unused-vars
				.map(([feature, _]) => Number(feature)),
			permissions: Object.fromEntries(entries.filter((e) => e.scope).map((e) => [e.scope, e.patterns]))
		})
	);
</script>

<Card class="mb-4 w-full items-start">
	<div>
		<h1 class="text-2xl font-semibold text-primary/90">{role.name}</h1>
		<p class="text-primary/60">{role.description}</p>
	</div>
	<div class="flex flex-row flex-wrap gap-1">
		{#if role.builtin}
			<StaticBadge variant="jam" text="builtin" />
		{/if}
		{#if role.scope}
			<StaticBadge variant="primary" text={`scope: ${role.scope}`} />
		{/if}
		<StaticBadge variant="primary" text={role.id} />
	</div>

	{#if role.builtin}
		<p class="text-sm text-primary/60 italic">This role is builtin; its metadata and permissions cannot be modified.</p>
	{:else}
		<div class="flex flex-col gap-2">
			{#if updateState.forbidden}
				<p class="text-sm text-primary/60 italic">You are not allowed to update this role.</p>
			{:else}
				<form
					class="flex max-w-160 flex-col gap-3"
					onsubmit={() =>
						Submit(async () => {
							await Glue.role.update(create(UpdateRequestSchema, { mod: mod }));
							refresh();
						}, updateState)}
				>
					<h2 class="text-xl font-semibold text-primary/80">Metadata</h2>
					<div class="flex flex-row items-center gap-2">
						<Input class="max-w-48" bind:value={name} placeholder="Name of the role" />
						<Input class="max-w-96" bind:value={description} placeholder="Description of the role" />
					</div>
					<p class="text-xs text-destructive">{Glue.Validate(RoleSchema, mod).violation.name ?? ''}</p>

					<h2 class="text-xl font-semibold text-primary/80">Features</h2>
					<FeatureEditor bind:features />

					<Button
						type="submit"
						variant="primary"
						class="mt-3 w-60"
						processing={updateState.loading}
						disabled={updateState.loading || Boolean(Glue.Validate(RoleSchema, mod).violation.name)}
					>
						Save Configuration
					</Button>
					{#if updateState.error}
						<Alert class="p-2" message={updateState.error} animation={true} />
					{/if}
				</form>
			{/if}
		</div>

		<Separator />

		<div class="flex flex-col gap-2">
			<h2 class="text-xl font-semibold text-primary/80">Permissions</h2>
			{#if configureState.forbidden}
				<p class="text-sm text-primary/60 italic">You are not allowed to configure this role.</p>
			{:else}
				<p class="text-sm text-primary/60">
					Comma separated glob patterns matched against the rpc procedure names, granted to subjects of the entries
					scope. You can only grant scopes you possess yourself.
				</p>
				<PermissionEditor bind:entries />
				<Button
					variant="primary"
					class="mt-3 w-60"
					processing={configureState.loading}
					disabled={configureState.loading || Boolean(Glue.Validate(RoleSchema, mod).violation.permissions)}
					onclick={() =>
						Submit(async () => {
							await Glue.rbac.configureRole(create(ConfigureRoleRequestSchema, { mod: mod }));
							refresh();
						}, configureState)}
				>
					Save Permissions
				</Button>
				{#if configureState.error}
					<Alert class="p-2" message={configureState.error} animation={true} />
				{/if}
			{/if}
		</div>

		<Separator />
	{/if}

	<div class="flex flex-col gap-2">
		<h2 class="text-xl font-semibold text-primary/80">Scope</h2>
		{#if attachState.forbidden}
			<p class="text-sm text-primary/60 italic">You are not allowed to attach scopes.</p>
		{:else}
			<p class="text-sm text-primary/60">Moves this role into another scope you possess.</p>
			<OptionalSelect
				bind:value={attachScope}
				placeholder="New scope"
				suggestions={getAssignableScopes().map((scope) => ({ id: scope, title: scope }))}
			/>
			<Button
				variant="primary"
				class="mt-3 w-60"
				processing={attachState.loading}
				disabled={attachState.loading || !attachScope}
				onclick={() =>
					Submit(async () => {
						await Glue.rbac.attachScope(
							create(AttachScopeRequestSchema, {
								resource: Resource.RoleData,
								id: role.id,
								scope: attachScope
							})
						);
						refresh();
					}, attachState)}
			>
				Modify Scope
			</Button>
			{#if attachState.error}
				<Alert class="p-2" message={attachState.error} animation={true} />
			{/if}
		{/if}
	</div>

	{#if !role.builtin}
		<Separator />

		<div class="flex flex-col gap-2">
			<h2 class="text-xl font-semibold text-primary/80">Danger Zone</h2>
			{#if removeState.forbidden}
				<p class="text-sm text-primary/60 italic">You are not allowed to delete this role.</p>
			{:else}
				<div class="mt-3 flex w-60 flex-row items-center gap-2">
					{#if confirmDelete}
						<Button
							variant="destructive"
							class="w-full"
							disabled={removeState.loading}
							onclick={() =>
								Submit(async () => {
									await Glue.role.delete(create(DeleteRequestSchema, { id: role.id }));
									deleted();
								}, removeState)}
						>
							Yes, delete {role.name}
						</Button>
						<Button variant="primary" onclick={() => (confirmDelete = false)}>Cancel</Button>
					{:else}
						<Button variant="destructive" class="w-full" onclick={() => (confirmDelete = true)}>Delete Role</Button>
					{/if}
				</div>
				{#if removeState.error}
					<Alert class="p-2" message={removeState.error} animation={true} />
				{/if}
			{/if}
		</div>
	{/if}
</Card>
