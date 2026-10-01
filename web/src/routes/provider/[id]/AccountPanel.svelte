<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import * as Alert from '$lib/components/shad/alert';
	import { Badge } from '$lib/components/shad/badge';
	import { Button } from '$lib/components/shad/button';
	import * as Card from '$lib/components/shad/card';
	import { Input } from '$lib/components/shad/input';
	import { Separator } from '$lib/components/shad/separator';
	import Spinner from '$lib/components/shad/spinner/spinner.svelte';
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
	import AlertCircleIcon from '@lucide/svelte/icons/alert-circle';

	let { account, refresh }: { account: Account; refresh: () => void } = $props();

	// the panel is remounted (keyed) per account, capturing the initial value is intended.
	// svelte-ignore state_referenced_locally
	let description = $state(account.description);
	let confirmDelete = $state(false);
	let force = $state(false);

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

	let updateState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let resetState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let credentialState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let fixState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let removeState: SubmitState = $state({ error: '', loading: false, forbidden: false });
</script>

<Card.Root class="w-full">
	<Card.Header>
		<Card.Title class="text-2xl">{account.name}</Card.Title>
		<Card.Description>{account.description}</Card.Description>
		<div class="flex flex-row flex-wrap gap-1">
			<Badge variant={account.state === AccountState.Corrupted ? 'destructive' : 'secondary'}>
				{AccountState[account.state]}
			</Badge>
			<Badge variant="outline">scope: {account.scope}</Badge>
			{#if account.targetId}
				<Badge variant="outline" class="font-mono">target: {account.targetId}</Badge>
			{/if}
			{#if account.boundUntil}
				<Badge variant="outline">bound until: {timestampDate(account.boundUntil).toLocaleString()}</Badge>
			{/if}
		</div>
	</Card.Header>
	<Card.Content class="flex flex-col gap-6">
		{#if account.error}
			<Alert.Root variant="destructive">
				<AlertCircleIcon />
				<Alert.Title>Provider reported an error</Alert.Title>
				<Alert.Description class="font-mono text-xs break-all">{account.error}</Alert.Description>
			</Alert.Root>
		{/if}

		<div class="flex flex-col gap-2">
			<Card.Title>Description</Card.Title>
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
					<Button type="submit" variant="outline" class="cursor-pointer" disabled={updateState.loading}>Save</Button>
				</form>
				{#if updateState.error}
					<p class="text-xs text-destructive">{updateState.error}</p>
				{/if}
			{/if}
		</div>

		<Separator />

		<div class="flex flex-col gap-2">
			<Card.Title>Maintenance</Card.Title>
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
						<Badge>
							<Spinner />
							Generating Credentials
						</Badge>
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
						<Badge>
							<Spinner />
							Initializing Eviction
						</Badge>
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
						<Badge>
							<Spinner />
							Force Reseting
						</Badge>
					{/if}
				</div>
				{#if fixState.error}
					<p class="text-xs text-destructive">{fixState.error}</p>
				{/if}
			{/if}
		</div>

		<Separator />

		<div class="flex flex-col gap-2">
			<Card.Title>Danger Zone</Card.Title>
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
	</Card.Content>
</Card.Root>
