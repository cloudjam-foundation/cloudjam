<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import StaticBadge from '$lib/components/custom/badges/StaticBadge.svelte';
	import StateBadge from '$lib/components/custom/badges/StateBadge.svelte';
	import Popup from '$lib/components/layout/Popup.svelte';
	import * as Alert from '$lib/components/shad/alert';
	import { Button } from '$lib/components/shad/button';
	import * as Dialog from '$lib/components/shad/dialog';
	import { Input } from '$lib/components/shad/input';
	import { Separator } from '$lib/components/shad/separator';
	import Spinner from '$lib/components/shad/spinner/spinner.svelte';
	import {
		AggregateCostRequestSchema,
		CredentialsRequestSchema,
		DeleteRequestSchema,
		FixRequestSchema,
		ResetRequestSchema,
		UpdateRequestSchema
	} from '$lib/sdk/v1/cloud/account/account_pb';
	import { AccountState, type Account } from '$lib/sdk/v1/cloud/account_pb';
	import { create } from '@bufbuild/protobuf';
	import { durationFromMs, timestampDate } from '@bufbuild/protobuf/wkt';
	import { CopyIcon, HandCoinsIcon, KeyRoundIcon, SquareArrowUpRightIcon } from '@lucide/svelte';
	import AlertCircleIcon from '@lucide/svelte/icons/alert-circle';
	import LoadingBadge from '$lib/components/custom/badges/LoadingBadge.svelte';
	import ProcessingButton from '$lib/components/custom/buttons/ProcessingButton.svelte';
	import Badge from '$lib/components/shad/badge/badge.svelte';

	let { account, open = $bindable(), refresh }: { account: Account; open: boolean; refresh: () => void } = $props();

	// the panel is remounted (keyed) per account, capturing the initial value is intended.
	// svelte-ignore state_referenced_locally
	let description = $state(account.description);
	let confirmDelete = $state(false);
	let force = $state(false);

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

	let costState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let updateState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let resetState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let credentialState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let fixState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let removeState: SubmitState = $state({ error: '', loading: false, forbidden: false });
</script>

<Popup bind:open title={account.name} description={account.description} {badges} {body} error={account.error} />

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
	<StaticBadge text={`scope: ${account.scope}`} />
	{#if account.targetId}
		<StaticBadge text={`target: ${account.targetId}`} />
	{/if}
	{#if account.boundUntil}
		<StaticBadge text={`bound until: ${timestampDate(account.boundUntil).toLocaleString()}`} />
	{/if}
{/snippet}

{#snippet body()}
	<div class="flex flex-col gap-2">
		<Dialog.Title>Description</Dialog.Title>
		{#if updateState.forbidden}
			<p class="text-sm text-muted-foreground italic">You are not allowed to update this account.</p>
		{:else}
			<p class="text-sm text-muted-foreground">Only accounts in the ready state accept metadata updates.</p>
			<form
				class="flex flex-row items-center gap-2"
				onsubmit={() =>
					Submit(async () => {
						await Glue.account.update(create(UpdateRequestSchema, { mod: { ...account, description: description } }));
						refresh();
					}, updateState)}
			>
				<Input class="max-w-96" bind:value={description} placeholder="Purpose of the account" />
				<Button type="submit" variant="outline" class="cursor-pointer">Save</Button>
				{#if updateState.loading}
					<Spinner />
				{/if}
			</form>
			{#if updateState.error}
				<p class="text-xs text-destructive">{updateState.error}</p>
			{/if}
		{/if}
	</div>

	<Separator />

	<div class="flex flex-col gap-2">
		<Dialog.Title>Cost & Spendings</Dialog.Title>
		{#if costState.forbidden}
			<p class="text-sm text-muted-foreground italic">You are not allowed to aggregate cost of this account.</p>
		{:else}
			<p class="text-sm text-muted-foreground">Aggregate account cost report</p>
			<div class="flex flex-row items-center gap-2">
				<ProcessingButton
					variant="primary"
					processing={costState.loading}
					hint={'Aggregating...'}
					onclick={() =>
						Submit(async () => {
							const resp = await Glue.account.aggregateCost(
								create(AggregateCostRequestSchema, {
									providerId: account.providerId,
									id: account.id,
									window: durationFromMs(6.048e8 * 20)
								})
							);

							for (const [name, service] of Object.entries(resp.services)) {
								for (const cost of service.cost) {
									totalCost += cost;
								}
							}
						}, costState)}
				>
					<HandCoinsIcon />
					Aggregate Cost
				</ProcessingButton>
				<p>Total Cost: {totalCost}$</p>
			</div>

			{#if costState.error}
				<Alert.Root variant="destructive">
					<AlertCircleIcon />
					<Alert.Title>Failed to aggregate cost</Alert.Title>
					<Alert.Description>{costState.error}</Alert.Description>
				</Alert.Root>
			{/if}
		{/if}

		<Separator />

		<div class="flex flex-col gap-2">
			<Dialog.Title>Maintenance</Dialog.Title>
			{#if fixState.forbidden}
				<p class="text-sm text-muted-foreground italic">You are not allowed to fix this account.</p>
			{:else}
				<p class="text-sm text-muted-foreground">Generates short lived credentials for the account.</p>
				<div class="flex flex-row items-center gap-2">
					<Button
						class="cursor-pointer self-start"
						disabled={credentialState.loading}
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
					{#if credentialState.loading}
						<LoadingBadge text={'Generating Credentials'} />
					{/if}
					{#if credentials}
						<Button
							variant="outline"
							class="cursor-pointer"
							onclick={() =>
								navigator.clipboard.writeText(
									parsedCredentials.env.length
										? parsedCredentials.env.map(([name, value]) => `export ${name}=${value}`).join('\n')
										: credentials
								)}
						>
							<CopyIcon /> Copy as environment
						</Button>
						<Button variant="secondary" class="cursor-pointer" href={parsedCredentials.url}>
							<SquareArrowUpRightIcon /> Open AWS Console
						</Button>
					{/if}
				</div>
				<p class="text-sm text-muted-foreground">
					Nukes all resources and prepares the account so that it can be used for challenges again.
				</p>
				<div class="flex flex-row items-center gap-2">
					<Button
						class="cursor-pointer self-start"
						variant="secondary"
						disabled={resetState.loading}
						onclick={() =>
							Submit(async () => {
								await Glue.account.reset(
									create(ResetRequestSchema, { providerId: account.providerId, id: account.id })
								);
								refresh();
							}, resetState)}
					>
						Reset Account
					</Button>
					{#if resetState.loading}
						<LoadingBadge text={'Initializing Eviction'} />
					{/if}
				</div>
				{#if resetState.error}
					<Alert.Root variant="destructive">
						<AlertCircleIcon />
						<Alert.Title>Failed to reset account</Alert.Title>
						<Alert.Description>{resetState.error}</Alert.Description>
					</Alert.Root>
				{/if}

				<p class="text-sm text-muted-foreground">
					Forces the account back into the ready state. Only do this after you actually repaired the account on the
					provider, otherwise it is handed out broken.
				</p>
				<div class="flex flex-row items-center gap-2">
					<Button
						variant="destructive"
						class="cursor-pointer self-start"
						disabled={fixState.loading}
						onclick={() =>
							Submit(async () => {
								await Glue.account.fix(create(FixRequestSchema, { providerId: account.providerId, id: account.id }));
								refresh();
							}, fixState)}
					>
						Force Metafix
					</Button>
					{#if fixState.loading}
						<LoadingBadge text={'Force Reseting'} />
					{/if}
				</div>
				{#if fixState.error}
					<p class="text-xs text-destructive">{fixState.error}</p>
				{/if}
			{/if}
		</div>

		<Separator />

		<div class="flex flex-col gap-2">
			<Dialog.Title>Danger Zone</Dialog.Title>
			{#if removeState.forbidden}
				<p class="text-sm text-muted-foreground italic">You are not allowed to delete this account.</p>
			{:else}
				<label class="flex flex-row items-center gap-2 text-sm text-muted-foreground">
					<input type="checkbox" bind:checked={force} class="cursor-pointer" />
					Force: drop the metadata immediately without waiting for the provider (the account leaks).
				</label>
				<div class="flex flex-row items-center gap-2">
					{#if confirmDelete}
						<Button
							variant="destructive"
							class="cursor-pointer"
							disabled={removeState.loading}
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
						<Button variant="outline" class="cursor-pointer" onclick={() => (confirmDelete = false)}>Cancel</Button>
					{:else}
						<Button variant="destructive" class="cursor-pointer" onclick={() => (confirmDelete = true)}>
							Delete Account
						</Button>
					{/if}
				</div>
				{#if removeState.error}
					<Alert.Root variant="destructive">
						<AlertCircleIcon />
						<Alert.Title>Failed to delete account</Alert.Title>
						<Alert.Description>{removeState.error}</Alert.Description>
					</Alert.Root>
				{/if}
			{/if}
		</div>
	</div>
{/snippet}
