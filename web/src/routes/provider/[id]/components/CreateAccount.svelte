<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import Card from '$lib/components/Card.svelte';
	import { CreateRequestSchema } from '$lib/sdk/v1/cloud/account/account_pb';
	import { AccountSchema } from '$lib/sdk/v1/cloud/account_pb';
	import { create } from '@bufbuild/protobuf';

	let { providerId, onCreated }: { providerId: string; onCreated: () => void } = $props();

	let createState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	let init = $state(create(AccountSchema, { id: crypto.randomUUID() }));

	let request = $derived(create(CreateRequestSchema, { init: { ...init, providerId: providerId } }));
</script>

<Card class="w-full items-start">
	<div>
		<h1 class="text-xl font-semibold text-primary/80">Provision Account</h1>
		<p class="text-primary/60">
			Creates a sandbox account on the provider. Provisioning runs asynchronously; the account joins the pool once it
			reaches the ready state.
		</p>
	</div>
	<form
		class="flex w-full flex-col gap-4"
		onsubmit={() =>
			Submit(async () => {
				await Glue.account.create(request);
				init = create(AccountSchema, { id: crypto.randomUUID() });
				onCreated();
			}, createState)}
	>
		<div class="grid gap-4 md:grid-cols-2">
			<Input
				bind:value={init.name}
				label="Name"
				placeholder="Name of the account"
				validation={Glue.Validate(AccountSchema, init).violation.name}
			/>
			<Input
				bind:value={init.description}
				label="Description"
				placeholder="Purpose of the account"
				validation={Glue.Validate(AccountSchema, init).violation.description}
			/>
		</div>
		<Button
			type="submit"
			variant="primary"
			class="self-start"
			processing={createState.loading}
			disabled={createState.loading || Boolean(Glue.Validate(CreateRequestSchema, request).error)}
		>
			Provision
		</Button>
		{#if createState.error}
			<Alert
				title={createState.forbidden ? 'Permission denied' : 'Failed to provision account'}
				message={createState.error}
				animation={true}
			/>
		{/if}
	</form>
</Card>
