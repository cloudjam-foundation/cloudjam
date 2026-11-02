<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import StateBadge from '$lib/components/StateBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import { ListRequestSchema } from '$lib/sdk/v1/cloud/account/account_pb';
	import { AccountState, type Account } from '$lib/sdk/v1/cloud/account_pb';
	import { create } from '@bufbuild/protobuf';
	import { timestampDate } from '@bufbuild/protobuf/wkt';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import XIcon from '@lucide/svelte/icons/x';
	import AccountPanel from './AccountPanel.svelte';
	import CreateAccount from './CreateAccount.svelte';

	let { providerId }: { providerId: string } = $props();

	let selected: Account | undefined = $state();
	let openSelected = $state(false);
	let creating = $state(false);
	// bumping the version remounts the table and therefore reloads the accounts.
	let version = $state(0);

	let accountsState: SubmitState = $state({ error: '', forbidden: false, loading: false });
</script>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<h2 class="text-xl opacity-80">Accounts</h2>
		<Button variant="primary" class="ml-auto" onclick={() => (creating = !creating)}>
			{#if creating}
				<XIcon /> Close Menu
			{:else}
				<PlusIcon /> Provision Account
			{/if}
		</Button>
	</div>

	{#if creating}
		<CreateAccount {providerId} oncreated={() => version++} />
	{/if}

	{#if accountsState.forbidden}
		<Alert title="Permission Denied" message="You are not allowed to list the accounts of this provider" />
	{:else}
		{#key version}
			<DataTable
				class="max-h-[80vh]"
				load={async (startAfter: Account | undefined) => {
					const accounts: Account[] = [];
					await Submit(async () => {
						const resp = await Glue.account.list(
							create(ListRequestSchema, { providerId: providerId, limit: 100, startAfter: startAfter?.id })
						);
						accounts.push(...resp.accounts);
					}, accountsState);
					return accounts;
				}}
				onRowClick={(account) => {
					selected = account;
					openSelected = true;
				}}
				loadState={accountsState}
				columns={[
					{ title: 'State', width: '110px', render: renderState },
					{ title: 'Name', width: '20%', render: renderName },
					{ title: 'Description', width: '30%', render: renderDescription },
					{ title: 'Target', width: '25%', render: renderTarget },
					{ title: 'Bound Until', width: '25%', render: renderBoundUntil }
				]}
			/>
		{/key}
		{#snippet renderState(account: Account)}
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
		{/snippet}
		{#snippet renderName(account: Account)}
			{account.name}
		{/snippet}
		{#snippet renderDescription(account: Account)}
			<span class="text-primary/60">{account.description}</span>
		{/snippet}
		{#snippet renderTarget(account: Account)}
			<span class="font-mono text-xs">{account.targetId}</span>
		{/snippet}
		{#snippet renderBoundUntil(account: Account)}
			{account.boundUntil ? timestampDate(account.boundUntil).toLocaleString() : ''}
		{/snippet}
	{/if}

	{#if selected}
		<AccountPanel
			bind:open={openSelected}
			account={selected}
			refresh={() => {
				openSelected = false;
				version++;
			}}
		/>
	{/if}
</div>
