<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/ui/Alert.svelte';
	import LoadingBadge from '$lib/components/badges/LoadingBadge.svelte';
	import StaticBadge from '$lib/components/badges/StaticBadge.svelte';
	import StateBadge from '$lib/components/badges/StateBadge.svelte';
	import Button from '$lib/components/buttons/Button.svelte';
	import LinkButton from '$lib/components/buttons/LinkButton.svelte';
	import Input from '$lib/components/inputs/Input.svelte';
	import Popup from '$lib/components/popovers/Popup.svelte';
	import Separator from '$lib/components/ui/Separator.svelte';
	import {
		CredentialsRequestSchema,
		DeleteRequestSchema,
		FixRequestSchema,
		ResetRequestSchema,
		UpdateRequestSchema
	} from '$lib/sdk/v1/cloud/account/account_pb';
	import { AccountState, type Account } from '$lib/sdk/v1/cloud/account_pb';
	import { create } from '@bufbuild/protobuf';
	import { timestampDate } from '@bufbuild/protobuf/wkt';
	import { CopyIcon, KeyRoundIcon, SquareArrowUpRightIcon } from '@lucide/svelte';

	let { account, open = $bindable(), refresh }: { account: Account; open: boolean; refresh: () => void } = $props();

	let description = $derived(account.description);
	let confirmDelete = $state(false);
	let force = $state(false);

	// TODO: Remove?
	let totalCost: number = $state(0);

	let credentials: string = $state('');

	let parsedCredentials: { env: [string, string][]; url: string } = $derived.by(() => {
		const parsed = JSON.parse(credentials);
		return {
			env: Object.entries(JSON.parse(credentials)).filter(
				([key, value]) => typeof value === 'string' && value && key !== 'URL'
			) as [string, string][],
			url: parsed.URL.toString()
		};
	});

	// TODO: Remove?
	let costState: SubmitState = $state({ status: 'idle' });
	let updateState: SubmitState = $state({ status: 'idle' });
	let resetState: SubmitState = $state({ status: 'idle' });
	let credentialState: SubmitState = $state({ status: 'idle' });
	let fixState: SubmitState = $state({ status: 'idle' });
	let removeState: SubmitState = $state({ status: 'idle' });
</script>

<Popup bind:open title={account.name} description={account.description} {badges} {body} />

{#snippet badges()}
	<StateBadge
		text={AccountState[account.state]}
		state={account.state.toString()}
		mapper={{
			[AccountState.Provisioning.toString()]: 'RunningPositive',
			[AccountState.Preparing.toString()]: 'RunningPositive',
			[AccountState.Evicting.toString()]: 'RunningPositive',
			[AccountState.Deleting.toString()]: 'RunningNegative',
			[AccountState.Corrupted.toString()]: 'StableNegative',
			[AccountState.Ready.toString()]: 'StablePositive',
			[AccountState.Running.toString()]: 'RunningWarning'
		}}
	/>
	<StaticBadge variant="primary" text={`tenant: ${account.scope}`} />
	{#if account.targetId}
		<StaticBadge variant="primary" text={`target: ${account.targetId}`} />
	{/if}
	{#if account.boundUntil}
		<StaticBadge variant="primary" text={`bound until: ${timestampDate(account.boundUntil).toLocaleString()}`} />
	{/if}
{/snippet}

{#snippet body()}
	{#if account.error}
		<Alert title="Account reported an error" message={account.error} />
	{/if}

	<div class="flex flex-col gap-2">
		<h1 class="text-xl font-semibold text-primary/80">Description</h1>
		{#if updateState.status === 'forbidden'}
			<p class="text-sm text-primary/60 italic">You are not allowed to update this account.</p>
		{:else}
			<p class="text-sm text-primary/60">Only accounts in the ready state accept metadata updates.</p>
			<form
				class="flex flex-row items-center gap-2"
				onsubmit={() =>
					Submit(async () => {
						await Glue.account.update(create(UpdateRequestSchema, { mod: { ...account, description: description } }));
						refresh();
					}, updateState)}
			>
				<Input class="max-w-96" bind:value={description} placeholder="Purpose of the account" />
				<Button type="submit" variant="primary" processing={updateState.status === 'loading'}>Save</Button>
			</form>
			{#if updateState.error}
				<Alert class="p-2" message={updateState.error} animation={true} />
			{/if}
		{/if}
	</div>

	<Separator />

	<div class="flex flex-col gap-2">
		<h1 class="text-xl font-semibold text-primary/80">Maintenance</h1>
		{#if fixState.status === 'forbidden'}
			<p class="text-sm text-primary/60 italic">You are not allowed to fix this account.</p>
		{:else}
			<p class="text-sm text-primary/60">Generates short lived credentials for the account.</p>
			<div class="flex flex-row flex-wrap items-center gap-2">
				<Button
					variant="primary"
					disabled={credentialState.status === 'loading'}
					onclick={() =>
						Submit(async () => {
							credentials = (
								await Glue.account.credentials(
									create(CredentialsRequestSchema, { providerId: account.providerId, id: account.id })
								)
							).credentials;
						}, credentialState)}
				>
					<KeyRoundIcon />
					Generate Credentials
				</Button>
				{#if credentialState.status === 'loading'}
					<LoadingBadge text="Generating Credentials" />
				{/if}
				{#if credentials}
					<!-- TODO: Fix -->
					<Button
						variant="primary"
						onclick={() =>
							navigator.clipboard.writeText(
								parsedCredentials.env.length
									? parsedCredentials.env.map(([name, value]) => `export ${name}=${value}`).join('\n')
									: credentials
							)}
					>
						<CopyIcon /> Copy as environment
					</Button>
					<LinkButton href={parsedCredentials.url}>
						<SquareArrowUpRightIcon /> Open AWS Console
					</LinkButton>
				{/if}
			</div>
			<p class="text-sm text-primary/60">
				Nukes all resources and prepares the account so that it can be used for challenges again.
			</p>
			<div class="flex flex-row items-center gap-2">
				<Button
					variant="primary"
					disabled={resetState.status === 'loading'}
					onclick={() =>
						Submit(async () => {
							await Glue.account.reset(create(ResetRequestSchema, { providerId: account.providerId, id: account.id }));
							refresh();
						}, resetState)}
				>
					Reset Account
				</Button>
				{#if resetState.status === 'loading'}
					<LoadingBadge text="Initializing Eviction" />
				{/if}
			</div>
			{#if resetState.error}
				<Alert class="p-2" title="Failed to reset account" message={resetState.error} animation={true} />
			{/if}

			<p class="text-sm text-primary/60">
				Forces the account back into the ready state. Only do this after you actually repaired the account on the
				provider, otherwise it is handed out broken.
			</p>
			<div class="flex flex-row items-center gap-2">
				<Button
					variant="destructive"
					disabled={fixState.status === 'loading'}
					onclick={() =>
						Submit(async () => {
							await Glue.account.fix(create(FixRequestSchema, { providerId: account.providerId, id: account.id }));
							refresh();
						}, fixState)}
				>
					Force Metafix
				</Button>
				{#if fixState.status === 'loading'}
					<LoadingBadge text="Force Resetting" />
				{/if}
			</div>
			{#if fixState.error}
				<Alert class="p-2" message={fixState.error} animation={true} />
			{/if}
		{/if}
	</div>

	<Separator />

	<div class="flex flex-col gap-2">
		<h1 class="text-xl font-semibold text-primary/80">Danger Zone</h1>
		{#if removeState.status === 'forbidden'}
			<p class="text-sm text-primary/60 italic">You are not allowed to delete this account.</p>
		{:else}
			<label class="flex flex-row items-center gap-2 text-sm text-primary/60">
				<input type="checkbox" bind:checked={force} class="cursor-pointer" />
				Force: drop the metadata immediately without waiting for the provider (the account leaks).
			</label>
			<div class="flex flex-row items-center gap-2">
				{#if confirmDelete}
					<Button
						variant="destructive"
						disabled={removeState.status === 'loading'}
						onclick={() =>
							Submit(async () => {
								await Glue.account.delete(
									create(DeleteRequestSchema, {
										providerId: account.providerId,
										id: account.id,
										force: force
									})
								);
								refresh();
							}, removeState)}
					>
						Yes, delete {account.name}
					</Button>
					<Button variant="primary" onclick={() => (confirmDelete = false)}>Cancel</Button>
				{:else}
					<Button variant="destructive" onclick={() => (confirmDelete = true)}>Delete Account</Button>
				{/if}
			</div>
			{#if removeState.error}
				<Alert class="p-2" title="Failed to delete account" message={removeState.error} animation={true} />
			{/if}
		{/if}
	</div>
{/snippet}
