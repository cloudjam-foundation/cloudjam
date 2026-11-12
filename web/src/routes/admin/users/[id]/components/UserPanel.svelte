<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import OptionalSelect from '$lib/components/OptionalSelect.svelte';
	import Card from '$lib/components/Card.svelte';
	import Select from '$lib/components/Select.svelte';
	import Separator from '$lib/components/Separator.svelte';
	import { getAssignableTenants } from '$lib/auth.svelte.js';
	import { AttachRoleRequestSchema, AttachScopeRequestSchema, Resource } from '$lib/sdk/v1/admin/rbac/rbac_pb';
	import { ListRequestSchema as ListRolesRequestSchema } from '$lib/sdk/v1/admin/role/role_pb';
	import type { Role } from '$lib/sdk/v1/admin/role_pb';
	import { DeleteRequestSchema, ResetPasswordRequestSchema, UpdateRequestSchema } from '$lib/sdk/v1/admin/user/user_pb';
	import type { User } from '$lib/sdk/v1/admin/user_pb';
	import { create } from '@bufbuild/protobuf';
	import { timestampFromDate } from '@bufbuild/protobuf/wkt';
	import { toSvg } from 'jdenticon';
	import { onMount } from 'svelte';
	import CopyableValue from '$lib/components/CopyableValue.svelte';

	let { user, refresh, deleted }: { user: User; refresh: () => void; deleted: () => void } = $props();

	let organization = $derived(user.organization);
	let role = $derived(user.role);
	let attachTenant = $derived(user.scope);
	let resetCode = $state('');
	let confirmDelete = $state(false);

	// roles are only used to resolve names and attach them to the user.
	let roles: Role[] = $state([]);
	let rolesState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	let updateState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let attachRoleState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let attachState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let resetState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let removeState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	onMount(() =>
		Submit(async () => {
			roles = (await Glue.role.list(create(ListRolesRequestSchema, { limit: 100 }))).roles;
		}, rolesState)
	);
</script>

<Card class="w-full items-start">
	<div class="flex flex-row items-center gap-4">
		<img
			alt="user profile"
			src={`data:image/svg+xml;base64,${btoa(toSvg(user.pubId, 60))}`}
			height="4rem"
			class="rounded-lg bg-primary/5"
		/>
		<div>
			<h1 class="text-2xl font-semibold text-primary/90">{user.username}</h1>
			<p class="text-primary/60">{user.email}</p>
		</div>
	</div>
	<div class="flex flex-row flex-wrap gap-1">
		{#if user.privileged}
			<StaticBadge variant="jam" text="privileged" />
		{/if}
		{#if user.scope}
			<StaticBadge variant="primary" text={`tenant: ${user.scope}`} />
		{/if}
		{#if user.role}
			<StaticBadge variant="primary" text={`role: ${roles.find((r) => r.id === user.role)?.name ?? user.role}`} />
		{/if}
		<StaticBadge variant="primary" text={user.id} />
	</div>

	{#if user.privileged}
		<p class="text-sm text-primary/60 italic">This user is privileged; role, scope and profile cannot be modified.</p>
	{:else}
		<div class="flex flex-col gap-2">
			<h2 class="text-xl font-semibold text-primary/80">Organization</h2>
			{#if updateState.forbidden}
				<Alert class="p-2" message="You are not allowed to update this user" animation={true} />
			{:else}
				<form
					class="flex flex-row items-center gap-2"
					onsubmit={() =>
						Submit(async () => {
							await Glue.user.update(create(UpdateRequestSchema, { mod: { ...user, organization: organization } }));
							refresh();
						}, updateState)}
				>
					<Input class="max-w-96" bind:value={organization} placeholder="Organization of the user" />
					<Button type="submit" variant="primary" processing={updateState.loading}>Save</Button>
				</form>
				{#if updateState.error}
					<Alert class="p-2" message={updateState.error} animation={true} />
				{/if}
			{/if}
		</div>

		<Separator />

		<div class="flex flex-col gap-2">
			<h2 class="text-xl font-semibold text-primary/80">Role</h2>
			{#if rolesState.forbidden || attachRoleState.forbidden}
				<Alert class="p-2" message="You are not allowed to modify the role of this user" animation={true} />
			{:else}
				<div class="flex flex-row items-center gap-2">
					<Select
						bind:value={role}
						class="w-96"
						placeholder="Select a role"
						items={roles.map((item) => ({ value: item.id, label: item.name }))}
					/>
					<Button
						variant="primary"
						processing={attachRoleState.loading}
						disabled={attachRoleState.loading || !role || role === user.role}
						onclick={() =>
							Submit(async () => {
								await Glue.rbac.attachRole(create(AttachRoleRequestSchema, { id: user.id, role: role }));
								refresh();
							}, attachRoleState)}
					>
						Attach
					</Button>
				</div>
				{#if attachRoleState.error}
					<Alert class="p-2" message={attachRoleState.error} animation={true} />
				{/if}
			{/if}
		</div>

		<Separator />

		<div class="flex flex-col gap-2">
			<h2 class="text-xl font-semibold text-primary/80">Tenant</h2>
			<p class="text-sm text-primary/60">Move the user to another tenant. This won't affect their permissions</p>
			<div class="flex flex-row items-center gap-2">
				<OptionalSelect
					bind:value={attachTenant}
					placeholder="Tenant the user is placed in"
					suggestions={getAssignableTenants().map((tenant) => ({ id: tenant, title: tenant }))}
				/>
				<Button
					variant="primary"
					processing={attachState.loading}
					disabled={attachState.loading || !attachTenant}
					onclick={() =>
						Submit(async () => {
							await Glue.rbac.attachScope(
								create(AttachScopeRequestSchema, {
									resource: Resource.UserData,
									id: user.id,
									scope: attachTenant
								})
							);
							await Glue.rbac.attachScope(
								create(AttachScopeRequestSchema, {
									resource: Resource.CredsData,
									id: user.email,
									scope: attachTenant
								})
							);
							refresh();
						}, attachState)}
				>
					Update
				</Button>
			</div>
			{#if attachState.forbidden}
				<Alert class="p-2" message="You are not allowed to move the user to the selected tenant" animation={true} />
			{:else if attachState.error}
				<Alert class="p-2" message={attachState.error} animation={true} />
			{/if}
		</div>

		<Separator />
	{/if}

	<div class="flex flex-col items-start gap-2">
		<h2 class="text-xl font-semibold text-primary/80">Reset Password</h2>
		{#if resetState.forbidden}
			<Alert class="p-2" message="You are not allowed to reset this users password" animation={true} />
		{:else if resetCode}
			<p class="text-sm text-primary/60">Share the reset link with the user:</p>
			<div class="flex flex-row items-center gap-2">
				<CopyableValue value={resetCode} descriptor="Registration code" />
			</div>
			<div class="flex flex-row items-center gap-2">
				<CopyableValue
					value={`${location.origin}/register?email=${encodeURIComponent(user.email)}&username=${encodeURIComponent(user.username)}&code=${encodeURIComponent(resetCode)}`}
					descriptor="Registration link"
				/>
			</div>
		{:else}
			<div class="flex flex-row items-center gap-2">
				<Button
					variant="primary"
					processing={resetState.loading}
					onclick={() =>
						Submit(async () => {
							const resp = await Glue.user.resetPassword(
								create(ResetPasswordRequestSchema, {
									email: user.email,
									expires: timestampFromDate(new Date(Date.now() + 24 * 60 * 60 * 1000)) // expires in 24 hours
								})
							);
							resetCode = resp.code;
						}, resetState)}
				>
					Create Reset Code
				</Button>
				<p class="text-sm text-primary/60">Creates a one time code (valid 24 hours) to reset the password.</p>
			</div>
			{#if resetState.error}
				<Alert class="p-2" message={resetState.error} animation={true} />
			{/if}
		{/if}
	</div>

	{#if !user.privileged}
		<Separator />

		<div class="flex flex-col gap-2">
			<h2 class="text-xl font-semibold text-primary/80">Danger Zone</h2>
			{#if removeState.forbidden}
				<Alert class="p-2" message="You are not allowed to delete this user" animation={true} />
			{:else}
				<div class="flex flex-row items-center gap-2">
					{#if confirmDelete}
						<Button
							variant="destructive"
							disabled={removeState.loading}
							onclick={() =>
								Submit(async () => {
									await Glue.user.delete(create(DeleteRequestSchema, { id: user.id }));
									deleted();
								}, removeState)}
						>
							Yes, delete {user.username}
						</Button>
						<Button variant="primary" onclick={() => (confirmDelete = false)}>Cancel</Button>
					{:else}
						<Button variant="destructive" onclick={() => (confirmDelete = true)}>Delete User</Button>
					{/if}
				</div>
				{#if removeState.error}
					<Alert class="p-2" message={removeState.error} animation={true} />
				{/if}
			{/if}
		</div>
	{/if}
</Card>
