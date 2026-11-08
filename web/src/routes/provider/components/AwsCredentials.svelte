<script lang="ts">
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import { Info } from '@lucide/svelte';
	import { EyeIcon, EyeOffIcon } from '@lucide/svelte';

	type credentials = { endpoint: string; region: string; access_key: string; secret_key: string };

	let { value = $bindable() }: { value: string } = $props();

	const uid = $props.id();

	function parse(raw: string): credentials {
		try {
			const parsed = JSON.parse(raw);
			return {
				endpoint: parsed.endpoint ?? '',
				region: parsed.region ?? '',
				access_key: parsed.access_key ?? '',
				secret_key: parsed.secret_key ?? ''
			};
		} catch {
			return { endpoint: '', region: '', access_key: '', secret_key: '' };
		}
	}

	let creds = $state(parse(value));
	let reveal = $state(false);

	$effect(() => {
		if (!creds.endpoint && !creds.region && !creds.access_key && !creds.secret_key) {
			value = '';
		} else {
			value = JSON.stringify({
				endpoint: creds.endpoint,
				region: creds.region,
				access_key: creds.access_key,
				secret_key: creds.secret_key
			});
		}
	});
</script>

<div class="grid gap-4 md:grid-cols-2">
	<div class="flex flex-col gap-1">
		<Input bind:value={creds.region} label="Region" placeholder="us-east-1" />
		<p class="text-xs text-primary/60">Region the organization api is called in.</p>
	</div>
	<div class="flex flex-col gap-1">
		<Input bind:value={creds.endpoint} label="Endpoint" placeholder="https://localhost:4566 (optional)" />
		<p class="text-xs text-primary/60">Only for emulators like FakeCloud, leave empty for real AWS.</p>
	</div>
	<Input bind:value={creds.access_key} label="Access Key ID" placeholder="AKIA..." />
	<div class="flex flex-col gap-1">
		<label for="{uid}-secret-key" class="text-sm">Secret Access Key</label>
		<div class="flex flex-row items-center gap-2">
			<Input
				id="{uid}-secret-key"
				type={reveal ? 'text' : 'password'}
				bind:value={creds.secret_key}
				placeholder="Secret access key of the user"
			/>
			<Button variant="ghost" title={reveal ? 'Hide' : 'Reveal'} onclick={() => (reveal = !reveal)}>
				{#if reveal}
					<EyeOffIcon />
				{:else}
					<EyeIcon />
				{/if}
			</Button>
		</div>
	</div>
</div>
<p class="flex flex-row items-center gap-1 text-xs text-primary/60">
	<Info size={14} />
	The specified credentials must have unrestricted AWS Organization access. Please only use this on fully blank AWS root accounts
	with ZERO workloads in it!
</p>
<p class="flex flex-row items-center gap-1 text-xs text-primary/60">
	<Info size={14} />
	You must create a dedicated user with "AdministratorAccess" for this. The AWS root account cannot assume other roles!
</p>
