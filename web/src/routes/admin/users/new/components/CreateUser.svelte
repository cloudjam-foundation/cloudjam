<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import OptionalSelect from '$lib/components/OptionalSelect.svelte';
	import Card from '$lib/components/Card.svelte';
	import Select from '$lib/components/Select.svelte';
	import { CreateRequestSchema } from '$lib/sdk/v1/admin/user/user_pb.ts';
	import { UserSchema } from '$lib/sdk/v1/admin/user_pb.ts';
	import { create } from '@bufbuild/protobuf';
	import { timestampFromDate } from '@bufbuild/protobuf/wkt';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import CopyableValue from '$lib/components/CopyableValue.svelte';

	let { scopes = [], oncreated }: { scopes?: string[]; oncreated: () => void } = $props();

	const expiries = [
		{ value: '1', label: '1 hour' },
		{ value: '24', label: '1 day' },
		{ value: '168', label: '7 days' }
	];

	let createState: SubmitState = $state({ error: '', forbidden: false, loading: false });

	let init = $state(create(UserSchema, {}));
	let expiry = $state('24');
	let code = $state('');
	let link = $state('');

	let request = $derived(
		create(CreateRequestSchema, {
			init: init,
			expires: timestampFromDate(new Date(Date.now() + Number(expiry) * 60 * 60 * 1000))
		})
	);
</script>

<Card class="w-full items-start">
	<div>
		<h1 class="text-xl font-semibold text-primary/80">Invite User</h1>
		<p class="text-primary/60">
			Creates a new user and returns a one time invitation code to complete the registration.
		</p>
	</div>
	{#if code}
		<div class="flex w-full flex-col items-center gap-2">
			<h2 class="text-xl font-semibold text-primary/80">Invitation created</h2>
			<p class="text-sm text-primary/60">Share the registration link (or the raw code) with the invited user:</p>
			<div class="flex flex-row items-center gap-2">
				<CopyableValue value={code} descriptor="Registration code" />
			</div>
			<div class="flex flex-row items-center gap-2">
				<CopyableValue value={link} descriptor="Registration link" />
			</div>
		</div>
	{:else}
		<form
			class="flex w-full flex-col gap-4"
			onsubmit={() =>
				Submit(async () => {
					const resp = await Glue.user.create(request);
					code = resp.code;
					link = `${location.origin}/register?email=${encodeURIComponent(init.email)}&username=${encodeURIComponent(init.username)}&code=${encodeURIComponent(resp.code)}`;
					oncreated();
				}, createState)}
		>
			<div class="grid gap-4 md:grid-cols-2">
				<Input
					bind:value={init.username}
					label="Username"
					placeholder="Username of the new user"
					validation={Glue.Validate(UserSchema, init).violation.username}
				/>
				<Input
					bind:value={init.email}
					label="Email"
					placeholder="Email of the new user"
					validation={Glue.Validate(UserSchema, init).violation.email}
				/>
				<Input
					bind:value={init.organization}
					label="Organization"
					placeholder="Organization of the new user"
					validation={Glue.Validate(UserSchema, init).violation.organization}
				/>
				<Input
					bind:value={init.description}
					label="Slogan"
					placeholder="Slogan for the new user"
					validation={Glue.Validate(UserSchema, init).violation.description}
				/>
				<label class="flex flex-col gap-1 text-sm">
					Scope
					<OptionalSelect
						bind:value={init.scope}
						placeholder="Scope the user is placed in"
						suggestions={scopes.map((scope) => ({ id: scope, title: scope }))}
					/>
				</label>
				<div class="flex flex-col gap-1">
					<span class="text-sm">Invitation expires in</span>
					<Select bind:value={expiry} class="w-full" items={expiries} />
				</div>
			</div>
			<Button
				type="submit"
				variant="primary"
				class="self-start"
				processing={createState.loading}
				disabled={createState.loading || Boolean(Glue.Validate(CreateRequestSchema, request).error)}
			>
				Invite
			</Button>
			{#if createState.error}
				<Alert
					title={createState.forbidden ? 'Permission denied' : 'Failed to invite user'}
					message={createState.error}
					animation={true}
				/>
			{/if}
		</form>
	{/if}
</Card>
