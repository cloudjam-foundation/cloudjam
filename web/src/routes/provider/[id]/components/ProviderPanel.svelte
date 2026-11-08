<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import LoadingBadge from '$lib/components/LoadingBadge.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import Card from '$lib/components/Card.svelte';
	import Separator from '$lib/components/Separator.svelte';
	import { DeleteRequestSchema, UpdateRequestSchema } from '$lib/sdk/v1/cloud/provider/provider_pb';
	import { ProviderType, type Provider } from '$lib/sdk/v1/cloud/provider_pb';
	import { create } from '@bufbuild/protobuf';
	import AwsCredentials from '../../components/AwsCredentials.svelte';

	let { provider, refresh, deleted }: { provider: Provider; refresh: () => void; deleted: () => void } = $props();

	// the panel is remounted (keyed) per provider, capturing the initial values is intended.
	// svelte-ignore state_referenced_locally
	let mod = $state({ ...provider });
	// svelte-ignore state_referenced_locally
	let regions = $state(provider.regions.join(', '));
	let confirmDelete = $state(false);

	let updateState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let removeState: SubmitState = $state({ error: '', loading: false, forbidden: false });
</script>

<Card class="w-full items-start">
	<div>
		<div class="flex flex-row items-center gap-2">
			<h1 class="text-2xl font-semibold text-primary/90">{provider.name}</h1>
			{#if updateState.loading}
				<LoadingBadge text="Reprovisioning" />
			{/if}
		</div>
		<p class="text-primary/60">{provider.description}</p>
	</div>
	<div class="flex flex-row flex-wrap gap-1">
		<StaticBadge variant="primary" text={ProviderType[provider.type]} />
		<StaticBadge variant="primary" text={`scope: ${provider.scope}`} />
		<StaticBadge variant="primary" text={provider.id} />
	</div>

	<div class="flex flex-col gap-2">
		<h2 class="text-xl font-semibold text-primary/80">Configuration</h2>
		{#if updateState.forbidden}
			<p class="text-sm text-primary/60 italic">You are not allowed to update this provider.</p>
		{:else}
			<form
				class="flex flex-col gap-4"
				onsubmit={() =>
					Submit(async () => {
						await Glue.provider.update(
							create(UpdateRequestSchema, {
								mod: {
									...mod,
									regions: regions
										.split(',')
										.map((region) => region.trim())
										.filter((region) => region)
								}
							})
						);
						refresh();
					}, updateState)}
			>
				<div class="grid gap-4 md:grid-cols-2">
					<Input bind:value={mod.name} label="Name" placeholder="Name of the provider" />
					<Input bind:value={mod.description} label="Description" placeholder="Purpose of the provider" />
					<Input type="email" bind:value={mod.email} label="Email" placeholder="Email used to provision new accounts" />
					<Input bind:value={regions} label="Regions" placeholder="eu-central-1, us-east-1" />
				</div>

				<div class="flex flex-col gap-2">
					<h2 class="text-sm font-medium">Credentials</h2>
					{#if provider.type === ProviderType.AWS}
						<AwsCredentials bind:value={mod.credentials} />
					{:else}
						<Input bind:value={mod.credentials} placeholder="Provider specific credentials" />
					{/if}
				</div>
				<Button type="submit" variant="primary" class="self-start" processing={updateState.loading}>
					Update and Reprovision
				</Button>
			</form>
			{#if updateState.error}
				<Alert class="p-2" message={updateState.error} animation={true} />
			{/if}
		{/if}
	</div>

	<Separator />

	<div class="flex flex-col gap-2">
		<h2 class="text-xl font-semibold text-primary/80">Danger Zone</h2>
		{#if removeState.forbidden}
			<p class="text-sm text-primary/60 italic">You are not allowed to delete this provider.</p>
		{:else}
			<p class="text-sm text-primary/60">
				Before deleting a provider please remove all accounts and challenge definitions.
			</p>
			<div class="flex flex-row items-center gap-2">
				{#if confirmDelete}
					<Button
						variant="destructive"
						disabled={removeState.loading}
						onclick={() =>
							Submit(async () => {
								await Glue.provider.delete(create(DeleteRequestSchema, { id: provider.id }));
								deleted();
							}, removeState)}
					>
						Yes, delete {provider.name}
					</Button>
					<Button variant="primary" onclick={() => (confirmDelete = false)}>Cancel</Button>
				{:else}
					<Button variant="destructive" onclick={() => (confirmDelete = true)}>Delete Provider</Button>
				{/if}
			</div>
			{#if removeState.error}
				<Alert class="p-2" message={removeState.error} animation={true} />
			{/if}
		{/if}
	</div>
</Card>
