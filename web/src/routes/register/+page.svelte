<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Glue, Submit, type SubmitState } from '$lib';
	import logo from '$lib/assets/favicon.svg';
	import Alert from '$lib/components/ui/Alert.svelte';
	import Button from '$lib/components/buttons/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import { RegisterRequestSchema } from '$lib/sdk/v1/auth/auth_pb';
	import { create } from '@bufbuild/protobuf';
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import SmartForm from '$lib/components/forms/SmartForm.svelte';
	import SmartField from '$lib/components/forms/SmartField.svelte';

	let registerRequest = $state(create(RegisterRequestSchema, {}));

	let registerState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	onMount(() => {
		registerRequest.email = page.url.searchParams.get('email') ?? '';
		registerRequest.username = page.url.searchParams.get('username') ?? '';
		registerRequest.code = page.url.searchParams.get('code') ?? '';
		history.replaceState(null, '', '/register');
	});
</script>

<svelte:head>
	<title>Register | CloudJam</title>
	<meta property="og:title" content="RegisteRegister | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full items-center justify-center">
	<Card class="mt-[10%] w-96">
		<SmartForm
			bind:submitState={registerState}
			class="flex w-full flex-col items-center gap-4"
			onsubmit={() =>
				Submit(async () => {
					await Glue.auth.register(registerRequest);
					await goto(resolve('/login'));
				}, registerState)}
		>
			<img alt="icon" src={logo} class="h-32" />
			<h1 class="text-4xl opacity-80">Registration</h1>
			<SmartField id="email" type="email" bind:value={registerRequest.email} placeholder="Please enter your email" />
			<SmartField
				id="registerCode"
				bind:value={registerRequest.code}
				placeholder="The registration code given to you"
			/>

			<hr class="w-full border border-neutral/40" />

			<SmartField
				id="username"
				bind:value={registerRequest.username}
				placeholder="Please enter the username you want"
			/>
			<SmartField
				id="password"
				type="password"
				bind:value={registerRequest.password}
				placeholder="Create a super secret password"
			/>
			<SmartField
				id="passwordConfirmation"
				type="password"
				bind:value={registerRequest.confirmPassword}
				placeholder="Confirm your super secret password"
			/>

			<Button
				class="w-full"
				type="submit"
				variant="primary"
				processing={registerState.loading}
				disabled={Boolean(Glue.Validate(RegisterRequestSchema, registerRequest).error)}
			>
				Register
			</Button>
			{#if registerState.error}
				<Alert title="Registration failed" message={registerState.error} animation={true} />
			{/if}
		</SmartForm>
	</Card>
</div>
