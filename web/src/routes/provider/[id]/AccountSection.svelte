<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import * as Alert from '$lib/components/shad/alert';
	import { Button } from '$lib/components/shad/button';
	import * as Table from '$lib/components/shad/table';
	import { ListRequestSchema } from '$lib/sdk/v1/cloud/account/account_pb';
	import { AccountState, type Account } from '$lib/sdk/v1/cloud/account_pb';
	import { create } from '@bufbuild/protobuf';
	import { timestampDate } from '@bufbuild/protobuf/wkt';
	import AlertCircleIcon from '@lucide/svelte/icons/alert-circle';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import { onMount } from 'svelte';
	import AccountPanel from './AccountPanel.svelte';
	import CreateAccount from './CreateAccount.svelte';
	import StateBadge from '$lib/components/custom/badges/StateBadge.svelte';

	let { providerId }: { providerId: string } = $props();

	const limit = 100;

	let accounts: Account[] = $state([]);
	let exhausted = $state(true);
	let dialog = $state(false);

	let selected: Account | undefined = $state();
	let creating = $state(false);

	let listState: SubmitState = $state({ error: '', forbidden: false, loading: false });

	let timeoutId: ReturnType<typeof setTimeout>;
	function load(startAfter?: string) {
		clearTimeout(timeoutId);
		async function poll() {
			Submit(async () => {
				const resp = await Glue.account.list(
					create(ListRequestSchema, { providerId: providerId, limit: limit, startAfter: startAfter })
				);
				accounts = startAfter ? [...accounts, ...resp.accounts] : resp.accounts;
				exhausted = resp.accounts.length < limit;
			}, listState);
			timeoutId = setTimeout(poll, 10000);
		}
		poll();
	}

	onMount(() => {
		load();
		return () => clearTimeout(timeoutId);
	});
</script>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<h2 class="text-xl opacity-80">Accounts</h2>
		<Button variant="outline" class="ml-auto cursor-pointer" onclick={() => (creating = !creating)}>
			<PlusIcon /> Provision Account
		</Button>
	</div>

	{#if creating}
		<CreateAccount {providerId} oncreated={() => load()} />
	{/if}

	{#if listState.forbidden}
		<Alert.Root>
			<AlertCircleIcon />
			<Alert.Title>Permission Denied</Alert.Title>
			<Alert.Description>You are not allowed to list the accounts of this provider</Alert.Description>
		</Alert.Root>
	{:else}
		{#if selected}
			<AccountPanel account={selected} bind:open={dialog} refresh={() => ((dialog = false), load())} />
		{/if}
		<Table.Root>
			<Table.Header>
				<Table.Row>
					<Table.Head class="w-24">State</Table.Head>
					<Table.Head>Name</Table.Head>
					<Table.Head>Description</Table.Head>
					<Table.Head>Target</Table.Head>
					<Table.Head>Bound Until</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each accounts as account (account.id)}
					<Table.Row class="cursor-pointer" onclick={() => ((selected = account), (dialog = true))}>
						<Table.Cell>
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
						</Table.Cell>
						<Table.Cell class="flex flex-row items-center gap-2 font-medium">
							{account.name}
						</Table.Cell>
						<Table.Cell>{account.description}</Table.Cell>
						<Table.Cell class="font-mono text-xs">{account.targetId}</Table.Cell>
						<Table.Cell>
							{account.boundUntil ? timestampDate(account.boundUntil).toLocaleString() : ''}
						</Table.Cell>
					</Table.Row>
				{:else}
					<Table.Row>
						<Table.Cell colspan={5}>
							<p class="text-muted-foreground p-4 text-sm italic">
								{listState.loading ? 'Loading accounts…' : 'No accounts provisioned on this provider yet.'}
							</p>
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
		{#if !exhausted}
			<Button
				variant="outline"
				class="cursor-pointer self-center"
				disabled={listState.loading}
				onclick={() => load(accounts.at(-1)?.id)}
			>
				Load More
			</Button>
		{/if}
	{/if}

	{#if listState.error}
		<Alert.Root variant="destructive">
			<AlertCircleIcon />
			<Alert.Title>Failed to load accounts</Alert.Title>
			<Alert.Description>{listState.error}</Alert.Description>
		</Alert.Root>
	{/if}
</div>
