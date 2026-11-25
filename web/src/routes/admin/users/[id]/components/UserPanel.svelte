<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/ui/Alert.svelte';
	import StaticBadge from '$lib/components/badges/StaticBadge.svelte';
	import Button from '$lib/components/buttons/Button.svelte';
	import Input from '$lib/components/inputs/Input.svelte';
	import OptionalSelect from '$lib/components/inputs/OptionalSelect.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Select from '$lib/components/inputs/Select.svelte';
	import Separator from '$lib/components/ui/Separator.svelte';
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
	let rolesState: SubmitState = $state({ status: 'idle' });

	let updateState: SubmitState = $state({ status: 'idle' });
	let attachRoleState: SubmitState = $state({ status: 'idle' });
	let attachState: SubmitState = $state({ status: 'idle' });
	let resetState: SubmitState = $state({ status: 'idle' });
	let removeState: SubmitState = $state({ status: 'idle' });

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
			{#if updateState.status === 'forbidden'}
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
					<Button
						type="submit"
						variant="primary"
						processing={updateState.status === 'loading'}
						buttonState={updateState.status === 'success' ? 'success' : 'idle'}
					>
						Save
					</Button>
				</form>
				{#if updateState.error}
					<Alert class="p-2" message={updateState.error} animation={true} />
				{/if}
			{/if}
		</div>

		<Separator />

		<div class="flex flex-col gap-2">
			<h2 class="text-xl font-semibold text-primary/80">Role</h2>
			{#if rolesState.status === 'forbidden' || attachRoleState.status === 'forbidden'}
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
						processing={attachRoleState.status === 'loading'}
						disabled={attachRoleState.status === 'loading' || !role || role === user.role}
						buttonState={attachRoleState.status === 'success' ? 'success' : 'idle'}
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
					processing={attachState.status === 'loading'}
					disabled={attachState.status === 'loading' || !attachTenant}
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
			{#if attachState.status === 'forbidden'}
				<Alert class="p-2" message="You are not allowed to move the user to the selected tenant" animation={true} />
			{:else if attachState.status === 'error'}
				<Alert class="p-2" message={attachState.error} animation={true} />
			{/if}
		</div>

		<Separator />
	{/if}

	<div class="flex flex-col items-start gap-2">
		<h2 class="text-xl font-semibold text-primary/80">Reset Password</h2>
		{#if resetState.status === 'forbidden'}
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
					processing={resetState.status === 'loading'}
					buttonState={resetState.status === 'success' ? 'success' : 'idle'}
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
			{#if removeState.status === 'forbidden'}
				<Alert class="p-2" message="You are not allowed to delete this user" animation={true} />
			{:else}
				<div class="flex flex-row items-center gap-2">
					{#if confirmDelete}
						<Button
							variant="destructive"
							disabled={removeState.status === 'loading'}
							buttonState={removeState.status === 'success' ? 'success' : 'idle'}
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
